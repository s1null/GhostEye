package database

import (
	"fmt"
	"log"
	"time"
)

// SaveTerminalSessionToDB 保存终端会话到数据库
func SaveTerminalSessionToDB(username, terminalID string, buffer []byte) error {
	// 检查会话是否存在
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM terminal_sessions WHERE username = ? AND terminal_id = ?",
		username, terminalID,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("Failed to check terminal session: %v", err)
	}
	
	if count > 0 {
		// 更新现有会话
		_, err = db.Exec(
			"UPDATE terminal_sessions SET buffer = ?, last_active = CURRENT_TIMESTAMP, active = 1 WHERE username = ? AND terminal_id = ?",
			buffer, username, terminalID,
		)
		if err != nil {
			return fmt.Errorf("Failed to update terminal session: %v", err)
		}
	} else {
		// 创建新会话
		_, err = db.Exec(
			"INSERT INTO terminal_sessions (username, terminal_id, buffer, active) VALUES (?, ?, ?, 1)",
			username, terminalID, buffer,
		)
		if err != nil {
			return fmt.Errorf("Failed to create terminal session: %v", err)
		}
	}
	
	return nil
}

// LoadTerminalSessionFromDB 从数据库加载终端会话
func LoadTerminalSessionFromDB(username, terminalID string) ([]byte, bool, error) {
	var buffer []byte
	var active int
	err := db.QueryRow(
		"SELECT buffer, active FROM terminal_sessions WHERE username = ? AND terminal_id = ?",
		username, terminalID,
	).Scan(&buffer, &active)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("Failed to load terminal session: %v", err)
	}
	
	return buffer, active == 1, nil
}

// SetTerminalSessionActive 设置终端会话活跃状态
func SetTerminalSessionActive(username, terminalID string, active bool) error {
	activeValue := 0
	if active {
		activeValue = 1
	}
	
	_, err := db.Exec(
		"UPDATE terminal_sessions SET active = ?, last_active = CURRENT_TIMESTAMP WHERE username = ? AND terminal_id = ?",
		activeValue, username, terminalID,
	)
	if err != nil {
		return fmt.Errorf("Failed to set terminal session active status: %v", err)
	}
	
	return nil
}

// GetUserTerminalSessions 获取用户的终端会话
func GetUserTerminalSessions(username string) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		"SELECT terminal_id, created_at, last_active, active FROM terminal_sessions WHERE username = ? ORDER BY last_active DESC",
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to query user terminal sessions: %v", err)
	}
	defer rows.Close()
	
	sessions := make([]map[string]interface{}, 0)
	for rows.Next() {
		var terminalID, createdAt, lastActive string
		var active int
		if err := rows.Scan(&terminalID, &createdAt, &lastActive, &active); err != nil {
			return nil, fmt.Errorf("Failed to scan user terminal session data: %v", err)
		}
		
		// 解析时间
		lastActiveTime, _ := time.Parse("2006-01-02 15:04:05", lastActive)
		createdAtTime, _ := time.Parse("2006-01-02 15:04:05", createdAt)
		
		sessions = append(sessions, map[string]interface{}{
			"terminal_id": terminalID,
			"created_at":  createdAt,
			"last_active": lastActive,
			"active":      active == 1,
			"age":         time.Since(lastActiveTime).String(),
			"duration":    lastActiveTime.Sub(createdAtTime).String(),
		})
	}
	
	return sessions, nil
}

// CleanupOldTerminalSessions 清理旧的终端会话
func CleanupOldTerminalSessions(days int) error {
	result, err := db.Exec(
		"DELETE FROM terminal_sessions WHERE active = 0 AND julianday('now') - julianday(last_active) > ?",
		days,
	)
	if err != nil {
		return fmt.Errorf("Failed to clean up old terminal sessions: %v", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	log.Printf("Cleaned up %d terminal sessions inactive for more than %d days", rowsAffected, days)
	
	return nil
}

// DeleteTerminalSessionFromDB 从数据库中完全删除终端会话
func DeleteTerminalSessionFromDB(username, terminalID string) error {
	result, err := db.Exec(
		"DELETE FROM terminal_sessions WHERE username = ? AND terminal_id = ?",
		username, terminalID,
	)
	if err != nil {
		return fmt.Errorf("Failed to delete terminal session from database: %v", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("No terminal session found to delete: %s", terminalID)
		return nil
	}
	
	log.Printf("Terminal session %s has been deleted from database", terminalID)
	return nil
} 
