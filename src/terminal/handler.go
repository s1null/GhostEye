package terminal

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
	
	"github.com/gorilla/websocket"
	
	"ghosteye/auth"

)

// WebSocket升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,  // 增加缓冲区大小
	WriteBufferSize: 4096,  // 增加缓冲区大小
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

// TerminalHandler 处理WebSocket终端连接
func TerminalHandler(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// 获取终端ID（如果存在）
	terminalID := r.URL.Query().Get("terminalId")
	if terminalID == "" {
		// 如果未提供终端ID，生成一个新的，格式与前端一致: term_timestamp_random
		timestamp := time.Now().UnixNano()
		random := rand.Intn(10000)
		terminalID = fmt.Sprintf("%d_%d", timestamp, random)
	}
	
	// 检查URL参数中是否有token
	tokens, ok := r.URL.Query()["token"]
	var username string
	var isAuthenticated bool
	
	// 首先尝试通过URL参数中的token认证
	if ok && len(tokens) > 0 {
		token := tokens[0]
		
		// 验证token并获取用户名
		var valid bool
		username, valid = auth.ValidateToken(token)
		if valid {
			isAuthenticated = true
			log.Printf("User %s authenticated successfully via URL token, %s", username, r.RemoteAddr)
		}
	}
	
	// 如果URL参数认证失败，返回错误
	if !isAuthenticated {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// 配置WebSocket升级器
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源
		},
	}
	
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v, %s", err, r.RemoteAddr)
		return
	}
	
	// 发送简短的欢迎消息
	welcomeMsg := struct {
		Type       string `json:"type"`
		TerminalID string `json:"terminalId"`
		Data       string `json:"data"`
	}{
		Type:       "welcome",
		TerminalID: terminalID,
		Data:       "Terminal connected\r\n",
	}
	welcomeBytes, _ := json.Marshal(welcomeMsg)
	conn.WriteMessage(websocket.TextMessage, welcomeBytes)
	
	// 发送认证成功响应
	successResp := struct {
		Type       string `json:"type"`
		TerminalID string `json:"terminalId"`
		Success    bool   `json:"success"`
		Token      string `json:"token,omitempty"`
	}{
		Type:       "auth",
		TerminalID: terminalID,
		Success:    true,
	}
	
	respBytes, _ := json.Marshal(successResp)
	conn.WriteMessage(websocket.TextMessage, respBytes)
	
	// 处理终端
	HandleUnixTerminal(conn, r.RemoteAddr, terminalID, username)
}
