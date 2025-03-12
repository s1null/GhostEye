package terminal

import (
	"fmt"
	"log"
	"syscall"
	"time"
	
	"ghosteye/database"
	"ghosteye/models"
)

// 获取用户的终端会话
func GetUserSessions(username string) map[string]*models.TerminalSession {
	models.TerminalSessionsMux.Lock()
	defer models.TerminalSessionsMux.Unlock()
	
	sessions, exists := models.TerminalSessions[username]
	if !exists {
		sessions = make(map[string]*models.TerminalSession)
		models.TerminalSessions[username] = sessions
	}
	
	return sessions
}

// 获取特定终端会话
func GetTerminalSession(username, terminalID string) *models.TerminalSession {
	models.TerminalSessionsMux.Lock()
	defer models.TerminalSessionsMux.Unlock()
	
	sessions, exists := models.TerminalSessions[username]
	if !exists {
		return nil
	}
	
	return sessions[terminalID]
}

// 保存终端会话
func SaveTerminalSession(username, terminalID string, session *models.TerminalSession) {
	models.TerminalSessionsMux.Lock()
	defer models.TerminalSessionsMux.Unlock()
	
	sessions, exists := models.TerminalSessions[username]
	if !exists {
		sessions = make(map[string]*models.TerminalSession)
		models.TerminalSessions[username] = sessions
	}
	
	sessions[terminalID] = session
}

// 移除终端会话
func RemoveTerminalSession(username, terminalID string) {
	models.TerminalSessionsMux.Lock()
	defer models.TerminalSessionsMux.Unlock()
	
	sessions, exists := models.TerminalSessions[username]
	if !exists {
		return
	}
	
	// 关闭会话
	session, exists := sessions[terminalID]
	if exists {
		// 关闭所有客户端连接
		session.ClientsMutex.Lock()
		for _, conn := range session.Clients {
			conn.Close()
		}
		session.ClientsMutex.Unlock()
		
		// 关闭PTY
		if session.Ptmx != nil {
			session.Ptmx.Close()
		}
		
		// 关闭标准输入
		if session.Stdin != nil {
			session.Stdin.Close()
		}
		
		// 发送完成信号
		close(session.Done)
	}
	
	// 从映射中删除
	delete(sessions, terminalID)
}

// 保存终端会话到数据库
func SaveSessionToDatabase(username string, terminalID string, session *models.TerminalSession) {
	// 获取缓冲区数据
	session.Mutex.Lock()
	buffer := session.Buffer.Data
	session.Mutex.Unlock()
	
	// 保存到数据库
	err := database.SaveTerminalSessionToDB(username, terminalID, buffer)
	if err != nil {
		log.Printf("Failed to save terminal session to database: %v", err)
	}
}

// 将终端会话标记为非活跃
func MarkSessionInactive(username string, terminalID string) {
	// 更新数据库中的活跃状态
	err := database.SetTerminalSessionActive(username, terminalID, false)
	if err != nil {
		log.Printf("Failed to mark terminal session as inactive: %v", err)
	}
}

// 启动会话清理器
func StartSessionCleaner() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			CleanInactiveSessions()
		}
	}()
}

// 清理非活跃会话
func CleanInactiveSessions() {
	// 清理内存中的非活跃会话
	models.TerminalSessionsMux.Lock()
	now := time.Now()
	
	for username, sessions := range models.TerminalSessions {
		for terminalID, session := range sessions {
			// 检查会话是否超过30分钟未活跃
			if now.Sub(session.LastActive) > 30*time.Minute {
				// 保存会话到数据库
				SaveSessionToDatabase(username, terminalID, session)
				
				// 标记为非活跃
				MarkSessionInactive(username, terminalID)
				
				// 关闭会话
				// 关闭所有客户端连接
				session.ClientsMutex.Lock()
				for _, conn := range session.Clients {
					conn.Close()
				}
				session.ClientsMutex.Unlock()
				
				// 关闭PTY
				if session.Ptmx != nil {
					session.Ptmx.Close()
				}
				
				// 关闭标准输入
				if session.Stdin != nil {
					session.Stdin.Close()
				}
				
				// 发送完成信号
				close(session.Done)
				
				// 从映射中删除
				delete(sessions, terminalID)
				
				log.Printf("Cleaned inactive terminal session %s for user %s", terminalID, username)
			}
		}
		
		// 如果用户没有会话，删除用户映射
		if len(sessions) == 0 {
			delete(models.TerminalSessions, username)
		}
	}
	
	models.TerminalSessionsMux.Unlock()
	
	// 清理数据库中的旧会话（超过7天的非活跃会话）
	err := database.CleanupOldTerminalSessions(7)
	if err != nil {
		log.Printf("Failed to clean old sessions from database: %v", err)
	}
}

// 列出终端会话
func ListTerminals(username string) []map[string]interface{} {
	// 从数据库获取会话列表
	sessions, err := database.GetUserTerminalSessions(username)
	if err != nil {
		log.Printf("Failed to get user terminal sessions: %v", err)
		return []map[string]interface{}{}
	}
	
	// 添加内存中的活跃会话
	models.TerminalSessionsMux.Lock()
	defer models.TerminalSessionsMux.Unlock()
	
	userSessions, exists := models.TerminalSessions[username]
	if exists {
		for terminalID, session := range userSessions {
			// 检查是否已经在列表中
			found := false
			for _, s := range sessions {
				if s["terminal_id"] == terminalID {
					found = true
					break
				}
			}
			
			if !found {
				// 添加到列表
				sessions = append(sessions, map[string]interface{}{
					"terminal_id": terminalID,
					"created_at":  session.Created.Format("2006-01-02 15:04:05"),
					"last_active": session.LastActive.Format("2006-01-02 15:04:05"),
					"active":      true,
					"age":         time.Since(session.LastActive).String(),
					"duration":    session.LastActive.Sub(session.Created).String(),
				})
			}
		}
	}
	
	return sessions
}

// KillTerminalSession 强制终止终端会话
func KillTerminalSession(username, terminalID string) error {
	models.TerminalSessionsMux.Lock()
	defer models.TerminalSessionsMux.Unlock()
	
	sessions, exists := models.TerminalSessions[username]
	if !exists {
		return fmt.Errorf("No terminal sessions exist for user %s", username)
	}
	
	session, exists := sessions[terminalID]
	if !exists {
		return fmt.Errorf("Terminal session %s does not exist", terminalID)
	}
	
	// 终止进程
	if session.Cmd != nil && session.Cmd.Process != nil {
		// 尝试先优雅地终止进程
		err := session.Cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Printf("Failed to send SIGTERM: %v, attempting SIGKILL", err)
			// If SIGTERM fails, use SIGKILL to force termination
			err = session.Cmd.Process.Kill()
			if err != nil {
				return fmt.Errorf("Failed to terminate process: %v", err)
			}
		}
		
		// 等待进程退出
		go func() {
			session.Cmd.Wait()
			log.Printf("Process for terminal session %s has been terminated", terminalID)
		}()
	}
	
	// 在这里完全从数据库中删除终端会话，而不是保存它
	err := database.DeleteTerminalSessionFromDB(username, terminalID)
	if err != nil {
		log.Printf("Failed to delete terminal session from database: %v", err)
	}
	
	// 关闭所有客户端连接
	session.ClientsMutex.Lock()
	for clientIP, conn := range session.Clients {
		log.Printf("Closing connection for client %s", clientIP)
		conn.Close()
	}
	session.ClientsMutex.Unlock()
	
	// 关闭PTY
	if session.Ptmx != nil {
		session.Ptmx.Close()
	}
	
	// 关闭标准输入
	if session.Stdin != nil {
		session.Stdin.Close()
	}
	
	// 发送完成信号
	close(session.Done)
	
	// 从映射中删除
	delete(sessions, terminalID)
	
	return nil
} 
