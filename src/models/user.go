package models

import (
	"sync"
)

// UserState 存储每个用户的状态
type UserState struct {
	IsRunning bool   // 是否有命令在运行
	Command   string // 当前运行的命令
	Mutex     sync.Mutex
}

// 全局的用户状态管理
var (
	UserStates    = make(map[string]*UserState)
	UserStatesMux sync.Mutex
	// 会话管理
	SessionTokens    = make(map[string]string) // token -> username
	SessionTokensMux sync.Mutex
)

// Response 是一个简单的结构体，用于响应客户请求
// Code: 0 表示成功，非0 表示错误
// Message 用于描述错误或成功信息
// Data 是额外响应数据
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// AuthRequest 认证请求结构
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserCommand 用户自定义命令
type UserCommand struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

// TerminalSession 数据库中的终端会话记录
type TerminalSessionRecord struct {
	ID         int    `json:"id"`
	TerminalID string `json:"terminal_id"`
	Username   string `json:"username"`
	Buffer     []byte `json:"buffer"`
	CreatedAt  string `json:"created_at"`
	LastActive string `json:"last_active"`
	Active     bool   `json:"active"`
} 