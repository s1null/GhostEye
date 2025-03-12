package database

import (
	"fmt"
	"log"
)

// AddUserCommand 添加用户命令
func AddUserCommand(username, name, command, description string) error {
	_, err := db.Exec(
		"INSERT INTO user_commands (username, name, command, description) VALUES (?, ?, ?, ?)",
		username, name, command, description,
	)
	if err != nil {
		return fmt.Errorf("Failed to add user command: %v", err)
	}
	
	log.Printf("Command %s for user %s added successfully", name, username)
	return nil
}

// UpdateUserCommand 更新用户命令
func UpdateUserCommand(username, name, command, description string) error {
	result, err := db.Exec(
		"UPDATE user_commands SET command = ?, description = ? WHERE username = ? AND name = ?",
		command, description, username, name,
	)
	if err != nil {
		return fmt.Errorf("Failed to update user command: %v", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get affected rows: %v", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("Command %s does not exist or does not belong to user %s", name, username)
	}
	
	log.Printf("Command %s for user %s updated successfully", name, username)
	return nil
}

// DeleteUserCommand 删除用户命令
func DeleteUserCommand(username, name string) error {
	result, err := db.Exec(
		"DELETE FROM user_commands WHERE username = ? AND name = ?",
		username, name,
	)
	if err != nil {
		return fmt.Errorf("Failed to delete user command: %v", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get affected rows: %v", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("Command %s does not exist or does not belong to user %s", name, username)
	}
	
	log.Printf("Command %s for user %s deleted successfully", name, username)
	return nil
}

// GetUserCommands 获取用户命令
func GetUserCommands(username string) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		"SELECT id, name, command, description, created_at FROM user_commands WHERE username = ? ORDER BY name",
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to query user commands: %v", err)
	}
	defer rows.Close()
	
	commands := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var name, command, description, createdAt string
		if err := rows.Scan(&id, &name, &command, &description, &createdAt); err != nil {
			return nil, fmt.Errorf("Failed to scan user command data: %v", err)
		}
		
		commands = append(commands, map[string]interface{}{
			"id":          id,
			"name":        name,
			"command":     command,
			"description": description,
			"created_at":  createdAt,
		})
	}
	
	return commands, nil
} 