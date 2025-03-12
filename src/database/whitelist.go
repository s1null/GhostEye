package database

import (
	"fmt"
	"log"
)

// IsIPInWhitelist 检查IP是否在白名单中
func IsIPInWhitelist(ip string) bool {
	// 如果白名单为空，允许所有IP
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM ip_whitelist").Scan(&count)
	if err != nil {
		log.Printf("Failed to query whitelist count: %v", err)
		return false
	}
	
	if count == 0 {
		return true
	}
	
	// 检查IP是否在白名单中
	var exists int
	err = db.QueryRow("SELECT COUNT(*) FROM ip_whitelist WHERE ip = ?", ip).Scan(&exists)
	if err != nil {
		log.Printf("Failed to query IP whitelist: %v", err)
		return false
	}
	
	return exists > 0
}

// AddIPToWhitelist 添加IP到白名单
func AddIPToWhitelist(ip, description string) error {
	_, err := db.Exec("INSERT INTO ip_whitelist (ip, description) VALUES (?, ?)", ip, description)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: ip_whitelist.ip" {
			// 更新描述
			_, err = db.Exec("UPDATE ip_whitelist SET description = ? WHERE ip = ?", description, ip)
			if err != nil {
				return fmt.Errorf("Failed to update IP whitelist description: %v", err)
			}
			log.Printf("IP %s is already in the whitelist, description updated", ip)
			return nil
		}
		return fmt.Errorf("Failed to add IP to whitelist: %v", err)
	}
	log.Printf("IP %s has been added to the whitelist", ip)
	return nil
}

// RemoveIPFromWhitelist 从白名单中移除IP
func RemoveIPFromWhitelist(ip string) error {
	result, err := db.Exec("DELETE FROM ip_whitelist WHERE ip = ?", ip)
	if err != nil {
		return fmt.Errorf("Failed to remove IP from whitelist: %v", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get affected rows: %v", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("IP %s is not in the whitelist", ip)
	}
	
	log.Printf("IP %s has been removed from the whitelist", ip)
	return nil
}

// GetAllWhitelistIPs 获取所有白名单IP
func GetAllWhitelistIPs() ([]map[string]string, error) {
	rows, err := db.Query("SELECT ip, description, created_at FROM ip_whitelist")
	if err != nil {
		return nil, fmt.Errorf("Failed to query whitelist IPs: %v", err)
	}
	defer rows.Close()
	
	ips := make([]map[string]string, 0)
	for rows.Next() {
		var ip, description, createdAt string
		if err := rows.Scan(&ip, &description, &createdAt); err != nil {
			return nil, fmt.Errorf("Failed to scan whitelist IP data: %v", err)
		}
		
		ips = append(ips, map[string]string{
			"ip":          ip,
			"description": description,
			"created_at":  createdAt,
		})
	}
	
	return ips, nil
} 