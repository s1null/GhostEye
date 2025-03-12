package database

import (
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/mattn/go-sqlite3"
)

// 全局数据库连接
var db *sql.DB

// InitDatabase 初始化数据库
func InitDatabase() error {
	var err error
	// 打开SQLite数据库连接
	db, err = sql.Open("sqlite3", "./ghosteye.db")
	if err != nil {
		return fmt.Errorf("Failed to open database: %v", err)
	}

	// 创建用户表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("Failed to create users table: %v", err)
	}

	// 创建IP白名单表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ip_whitelist (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("Failed to create IP whitelist table: %v", err)
	}

	// 创建用户命令表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			name TEXT NOT NULL,
			command TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(username, name)
		)
	`)
	if err != nil {
		return fmt.Errorf("Failed to create user commands table: %v", err)
	}

	// 创建终端会话表，用于持久化终端会话
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS terminal_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			terminal_id TEXT NOT NULL,
			username TEXT NOT NULL,
			buffer BLOB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			active INTEGER DEFAULT 1,
			UNIQUE(username, terminal_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("Failed to create terminal sessions table: %v", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() {
	if db != nil {
		db.Close()
		log.Println("Database connection closed")
	}
}

// GetDB 获取数据库连接
func GetDB() *sql.DB {
	return db
} 