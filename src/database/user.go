package database

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// AddUser 添加用户
func AddUser(username, password string) error {
	_, err := db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("Username %s already exists", username)
		}
		return fmt.Errorf("Failed to add user: %v", err)
	}
	log.Printf("User %s added successfully", username)
	return nil
}

// ValidateUser 验证用户
func ValidateUser(username, password string) bool {
	var storedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&storedPassword)
	if err != nil {
		log.Printf("Failed to query user: %v", err)
		return false
	}
	
	return password == storedPassword
}

// GenerateStrongPassword 生成强密码
func GenerateStrongPassword(length int) string {
	if length < 8 {
		length = 8 // 最小长度为8
	}
	
	// 字符集
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	symbols := "!@#$%^&*()-_=+[]{}|;:,.<>?"
	
	// 确保至少包含每种字符
	password := make([]byte, length)
	
	// 随机数生成器
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	// 确保至少有一个小写字母
	password[0] = lowercase[r.Intn(len(lowercase))]
	
	// 确保至少有一个大写字母
	password[1] = uppercase[r.Intn(len(uppercase))]
	
	// 确保至少有一个数字
	password[2] = numbers[r.Intn(len(numbers))]
	
	// 确保至少有一个特殊符号
	password[3] = symbols[r.Intn(len(symbols))]
	
	// 填充剩余字符
	allChars := lowercase + uppercase + numbers + symbols
	for i := 4; i < length; i++ {
		password[i] = allChars[r.Intn(len(allChars))]
	}
	
	// 打乱顺序
	for i := range password {
		j := r.Intn(i + 1)
		password[i], password[j] = password[j], password[i]
	}
	
	return string(password)
}

// GenerateRandomUsername 生成随机用户名
func GenerateRandomUsername(prefix string, length int) string {
	if length < 4 {
		length = 4 // 最小长度为4
	}
	
	// 字符集
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	
	// 随机数生成器
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	// 生成随机用户名
	username := make([]byte, length)
	
	// 确保至少有一个小写字母
	username[0] = lowercase[r.Intn(len(lowercase))]
	
	// 确保至少有一个大写字母
	username[1] = uppercase[r.Intn(len(uppercase))]
	
	// 确保至少有一个数字
	username[2] = numbers[r.Intn(len(numbers))]
	
	// 填充剩余字符
	allChars := lowercase + uppercase + numbers
	for i := 3; i < length; i++ {
		username[i] = allChars[r.Intn(len(allChars))]
	}
	
	// 打乱顺序
	for i := range username {
		j := r.Intn(i + 1)
		username[i], username[j] = username[j], username[i]
	}
	
	// 添加前缀
	if prefix != "" {
		return prefix + string(username)
	}
	
	return string(username)
}

// GenerateRandomUsers 生成随机用户
func GenerateRandomUsers(count int) []map[string]string {
	users := make([]map[string]string, 0, count)
	
	for i := 0; i < count; i++ {
		username := GenerateRandomUsername("user_", 16)
		password := GenerateStrongPassword(16)
		
		err := AddUser(username, password)
		if err != nil {
			log.Printf("Failed to generate random user: %v", err)
			continue
		}
		
		users = append(users, map[string]string{
			"username": username,
			"password": password,
		})
	}
	
	return users
}

// GetAllUsers 获取所有用户
func GetAllUsers() ([]map[string]string, error) {
	rows, err := db.Query("SELECT username, password FROM users")
	if err != nil {
		return nil, fmt.Errorf("Failed to query users: %v", err)
	}
	defer rows.Close()
	
	users := make([]map[string]string, 0)
	for rows.Next() {
		var username, password string
		if err := rows.Scan(&username, &password); err != nil {
			return nil, fmt.Errorf("Failed to scan user data: %v", err)
		}
		
		users = append(users, map[string]string{
			"username": username,
			"password": password,
		})
	}
	
	return users, nil
} 