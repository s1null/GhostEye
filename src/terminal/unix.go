package terminal

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"
	"encoding/json"
	"encoding/base64"
	
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	
	"ghosteye/database"
	"ghosteye/models"
)

// 处理Unix终端会话
func HandleUnixTerminal(conn *websocket.Conn, clientIP string, terminalID string, username string) {
	// 创建新会话或获取现有会话
	session := GetTerminalSession(username, terminalID)
	
	// 如果会话存在且有活跃的PTY，直接连接到现有会话
	if session != nil && (session.Ptmx != nil || session.Cmd != nil) {
		log.Printf("Client %s connected to existing session %s", clientIP, terminalID)
		
		// 将客户端添加到会话中
		session.ClientsMutex.Lock()
		session.Clients[clientIP] = conn
		session.ClientsMutex.Unlock()
		
		// 更新会话活跃时间
		session.LastActive = time.Now()
		
		// 向客户端发送缓冲数据
		if len(session.Buffer.Data) > 0 {
			conn.WriteMessage(websocket.BinaryMessage, session.Buffer.Data)
		}
		
		// 处理WebSocket连接
		handleWebSocketConnection(conn, clientIP, session, username, terminalID)
		return
	}
	
	// 会话不存在或已关闭，尝试从数据库恢复或创建新会话
	if session == nil {
		// 尝试从数据库加载会话
		buffer, _, err := database.LoadTerminalSessionFromDB(username, terminalID)
		if err == nil && buffer != nil {
			// Recover session from database
			log.Printf("Recovering terminal session %s for user %s from database", terminalID, username)
			session = &models.TerminalSession{
				Done:       make(chan struct{}),
				LastActive: time.Now(),
				Clients:    make(map[string]*websocket.Conn),
				Buffer: models.OutputBuffer{
					Data: buffer,
					Max:  100 * 1024, // 最大100KB
				},
				Active:     true,
				Created:    time.Now(),
			}
			SaveTerminalSession(username, terminalID, session)
		} else {
			// 创建新会话
			session = &models.TerminalSession{
				Done:       make(chan struct{}),
				LastActive: time.Now(),
				Clients:    make(map[string]*websocket.Conn),
				Buffer: models.OutputBuffer{
					Data: []byte{},
					Max:  100 * 1024, // 最大100KB
				},
				Active:     true,
				Created:    time.Now(),
			}
			SaveTerminalSession(username, terminalID, session)
		}
	}

	// 将客户端添加到会话中
	session.ClientsMutex.Lock()
	session.Clients[clientIP] = conn
	session.ClientsMutex.Unlock()
	
	log.Printf("Client %s connected to session %s, current client count: %d", clientIP, terminalID, len(session.Clients))

	// 更新会话活跃时间
	session.LastActive = time.Now()

	// 向客户端发送缓冲数据
	if len(session.Buffer.Data) > 0 {
		conn.WriteMessage(websocket.BinaryMessage, session.Buffer.Data)
		conn.WriteMessage(websocket.BinaryMessage, []byte("\r\n--- History ends, new session begins ---\r\n"))
	}

	// 创建命令
	cmd := exec.Command("/bin/bash")
	
	// 创建伪终端
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v, %s", err, clientIP)
		conn.WriteMessage(websocket.BinaryMessage, []byte(fmt.Sprintf("Failed to start terminal: %v\r\n", err)))
		return
	}
	
	// 将PTY和命令保存到会话中
	session.Ptmx = ptmx
	session.Cmd = cmd
	
	// 设置终端大小
	pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	// 创建用于取消goroutine的上下文
	ctx, cancel := context.WithCancel(context.Background())
	session.CancelFunc = cancel
	
	// 创建WaitGroup等待所有goroutine
	var wg sync.WaitGroup

	// 从PTY读取并写入WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 设置读取超时以避免阻塞
				ptmx.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF && !os.IsTimeout(err) {
						log.Printf("Failed to read from PTY: %v, %s", err, clientIP)
					}
					// 超时错误是正常的，继续读取
					if os.IsTimeout(err) {
						continue
					}
					// 其他错误，退出循环
					return
				}
				
				if n > 0 {
					// 将输出添加到缓冲区
					session.Buffer.Append(buf[:n])
					
					// 构造带终端ID的消息
					message := struct {
						Type       string `json:"type"`
						TerminalID string `json:"terminalId"`
						Data       string `json:"data"`
					}{
						Type:       "output",
						TerminalID: terminalID,
						// 使用Base64编码二进制数据
						Data:       base64.StdEncoding.EncodeToString(buf[:n]),
					}
					
					// 序列化消息
					messageBytes, err := json.Marshal(message)
					if err != nil {
						log.Printf("Failed to serialize terminal output message: %v", err)
						continue
					}
					
					// 向所有客户端发送输出
					session.ClientsMutex.Lock()
					for clientAddr, clientConn := range session.Clients {
						err := clientConn.WriteMessage(websocket.TextMessage, messageBytes)
						if err != nil {
							log.Printf("Failed to send message to client %s: %v", clientAddr, err)
							// Don't remove client here to avoid concurrent map modification
						}
					}
					session.ClientsMutex.Unlock()
				}
			}
		}
	}()

	// 处理WebSocket连接
	handleWebSocketConnection(conn, clientIP, session, username, terminalID)
	
}

// 处理WebSocket连接
func handleWebSocketConnection(conn *websocket.Conn, clientIP string, session *models.TerminalSession, username string, terminalID string) {
	// 从WebSocket读取并写入PTY
	for {
		// 设置较长的读取超时以避免频繁超时
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			// 重置读取超时
			conn.SetReadDeadline(time.Time{})
			
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Client %s closed connection normally", clientIP)
			} else {
				log.Printf("Failed to read WebSocket message: %v, %s", err, clientIP)
			}
			
			// 从会话中移除客户端
			session.ClientsMutex.Lock()
			delete(session.Clients, clientIP)
			clientCount := len(session.Clients)
			session.ClientsMutex.Unlock()
			
			log.Printf("Client %s disconnected, %d clients remaining", clientIP, clientCount)
			
			// 如果没有更多客户端但命令仍在运行，保持会话活跃
			if clientCount == 0 {
				log.Printf("Command still running for session %s, keeping active", terminalID)
				// Save session to database
				SaveSessionToDatabase(username, terminalID, session)
			}
			
			return
		}
		
		// 重置读取超时
		conn.SetReadDeadline(time.Time{})
		
		// 处理不同类型的消息
		if messageType == websocket.TextMessage {
			// 尝试解析为JSON
			var message models.Message
			if err := json.Unmarshal(p, &message); err == nil {
				if message.Type == "resize" {
					// 处理终端大小调整
					if resizeData, ok := message.Data.(map[string]interface{}); ok {
						if cols, ok := resizeData["cols"].(float64); ok {
							if rows, ok := resizeData["rows"].(float64); ok {
								// 调整终端大小
								if session.Ptmx != nil {
									pty.Setsize(session.Ptmx, &pty.Winsize{
										Rows: uint16(rows),
										Cols: uint16(cols),
									})
								}
							}
						}
					}
					continue
				} else if message.Type == "heartbeat" {
					// 处理心跳消息
					// 更新会话活跃时间
					session.LastActive = time.Now()
					
					// 发送心跳响应
					heartbeatResp := struct {
						Type       string `json:"type"`
						TerminalID string `json:"terminalId"`
						Data       string `json:"data"`
					}{
						Type:       "heartbeat",
						TerminalID: terminalID,
						Data:       "pong",
					}
					
					respBytes, _ := json.Marshal(heartbeatResp)
					
					// 延迟心跳响应以避免过于频繁的通信
					time.Sleep(200 * time.Millisecond)
					
					if err := conn.WriteMessage(websocket.TextMessage, respBytes); err != nil {
						// Don't disconnect on heartbeat response failure, just log error
						log.Printf("Failed to send heartbeat response: %v, %s", err, clientIP)
					}
					continue
				} else if message.Type == "close" {
					// Handle close command
					log.Printf("Received close command from client %s", clientIP)
					
					// 从会话中移除客户端
					session.ClientsMutex.Lock()
					delete(session.Clients, clientIP)
					clientCount := len(session.Clients)
					session.ClientsMutex.Unlock()
					
					// 如果没有更多客户端，关闭会话
					if clientCount == 0 {
						// 将会话保存到数据库
						SaveSessionToDatabase(username, terminalID, session)
						
						// 如果用户明确请求关闭，终止进程
						if session.Cmd != nil && session.Cmd.Process != nil {
							log.Printf("Terminating command for session %s", terminalID)
							session.Cmd.Process.Signal(syscall.SIGTERM)
							time.Sleep(100 * time.Millisecond)
							session.Cmd.Process.Kill()
						}
						
						// 标记为非活跃
						MarkSessionInactive(username, terminalID)
						
						// 移除会话
						RemoveTerminalSession(username, terminalID)
					}
					
					return
				}
			}
			
			// 其他文本消息，写入PTY
			if session.Ptmx != nil {
				_, err = session.Ptmx.Write(p)
				if err != nil {
					log.Printf("Failed to write to PTY: %v, %s", err, clientIP)
				}
			}
		} else if messageType == websocket.BinaryMessage {
			// 二进制消息，直接写入PTY
			if session.Ptmx != nil {
				_, err = session.Ptmx.Write(p)
				if err != nil {
					log.Printf("Failed to write to PTY: %v, %s", err, clientIP)
				}
			}
		}
		
		// 更新会话活跃时间
		session.LastActive = time.Now()
	}
}

// 创建新会话
func createNewSession(conn *websocket.Conn, remoteAddr string, username string, terminalID string) *models.TerminalSession {
	// 创建新会话
	session := &models.TerminalSession{
		Done:         make(chan struct{}),
		LastActive:   time.Now(),
		Clients:      make(map[string]*websocket.Conn),
		Buffer:       models.OutputBuffer{Max: 100 * 1024}, // 100KB缓冲区
		Active:       true,
		Created:      time.Now(),
	}
	
	// 添加客户端
	session.ClientsMutex.Lock()
	session.Clients[remoteAddr] = conn
	session.ClientsMutex.Unlock()
	
	// 保存会话
	SaveTerminalSession(username, terminalID, session)
	
	return session
}

// 启动终端命令
func startTerminalCommand(session *models.TerminalSession, conn *websocket.Conn, remoteAddr string, username string, terminalID string) {
	// 确定shell命令
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	
	// 创建命令
	cmd := exec.Command(shell, "-l") // 使用登录shell
	
	// 设置环境变量
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	
	var err error
	
	// 创建PTY
	session.Ptmx, err = pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v", err)
		conn.WriteMessage(websocket.BinaryMessage, []byte("Failed to start terminal: "+err.Error()+"\r\n"))
		conn.Close()
		return
	}
	session.IsStandardPipe = false
	
	// 保存命令
	session.Cmd = cmd
	
	// 启动读取goroutine
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := session.Ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Failed to read from PTY: %v", err)
				}
				break
			}
			
			if n > 0 {
				data := buf[:n]
				
				// 更新最后活跃时间
				session.LastActive = time.Now()
				
				// 添加到缓冲区
				session.Mutex.Lock()
				session.Buffer.Append(data)
				session.Mutex.Unlock()
				
				// 广播给所有客户端
				session.ClientsMutex.Lock()
				for _, client := range session.Clients {
					client.WriteMessage(websocket.BinaryMessage, data)
				}
				session.ClientsMutex.Unlock()
			}
		}
		
		// PTY closed, close session
		log.Printf("PTY closed, terminating session %s", terminalID)
		
		// 将会话保存到数据库
		SaveSessionToDatabase(username, terminalID, session)
		
		// 标记为非活跃
		MarkSessionInactive(username, terminalID)
		
		// 移除会话
		RemoveTerminalSession(username, terminalID)
	}()
}

// 连接到现有会话
func ConnectToExistingSession(session *models.TerminalSession, conn *websocket.Conn, remoteAddr string) {
	log.Printf("Client %s connected to existing session", remoteAddr)
	
	// 添加客户端
	session.ClientsMutex.Lock()
	session.Clients[remoteAddr] = conn
	session.ClientsMutex.Unlock()
	
	// 发送缓冲数据
	session.Mutex.Lock()
	if len(session.Buffer.Data) > 0 {
		conn.WriteMessage(websocket.BinaryMessage, session.Buffer.Data)
	}
	session.Mutex.Unlock()
	
	// 启动WebSocket读取goroutine
	go func() {
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Failed to read WebSocket message: %v, %s", err, remoteAddr)
				break
			}
			
			// 更新最后活跃时间
			session.LastActive = time.Now()
			
			if messageType == websocket.TextMessage {
				// 处理文本消息（JSON控制消息）
				var message models.Message
				if err := json.Unmarshal(p, &message); err == nil {
					if message.Type == "resize" {
						// 处理终端大小调整
						if resizeData, ok := message.Data.(map[string]interface{}); ok {
							cols, _ := resizeData["cols"].(float64)
							rows, _ := resizeData["rows"].(float64)
							
							// 调整终端大小
							if !session.IsStandardPipe && session.Ptmx != nil {
								ResizeTerminal(session.Ptmx, int(cols), int(rows))
							}
						}
					} else if message.Type == "heartbeat" {
						// 处理心跳消息
						heartbeatResp := models.Message{
							Type: "heartbeat",
							Data: "pong",
						}
						respBytes, _ := json.Marshal(heartbeatResp)
						conn.WriteMessage(websocket.TextMessage, respBytes)
					}
				}
			} else if messageType == websocket.BinaryMessage {
				// 处理二进制消息（终端输入）
				if !session.IsStandardPipe && session.Ptmx != nil {
					session.Ptmx.Write(p)
				}
			}
		}
		
		// WebSocket关闭，移除客户端
		session.ClientsMutex.Lock()
		delete(session.Clients, remoteAddr)
		clientCount := len(session.Clients)
		session.ClientsMutex.Unlock()
		
		log.Printf("Client %s disconnected, %d clients remaining", remoteAddr, clientCount)
	}()
}

// 调整终端大小
func ResizeTerminal(ptmx *os.File, cols, rows int) {
	ws := struct {
		Rows    uint16
		Cols    uint16
		XPixel  uint16
		YPixel  uint16
	}{
		Rows:   uint16(rows),
		Cols:   uint16(cols),
		XPixel: 0,
		YPixel: 0,
	}
	
	// TIOCSWINSZ是设置终端窗口大小的ioctl命令
	const TIOCSWINSZ = 0x5414
	
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		ptmx.Fd(),
		uintptr(TIOCSWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	
	if errno != 0 {
		log.Printf("Failed to resize terminal: %v", errno)
	}
} 
