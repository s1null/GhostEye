package models

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message WebSocket消息结构
type Message struct {
	Type       string      `json:"type"`
	TerminalID string      `json:"terminalId"`  // 添加终端ID字段
	Data       interface{} `json:"data"`
}

// AuthResponse 认证响应结构
type AuthResponse struct {
	Type       string `json:"type"`
	TerminalID string `json:"terminalId"`  // 添加终端ID字段
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Token      string `json:"token,omitempty"`
}

// OutputBuffer 输出缓冲区
type OutputBuffer struct {
	Data []byte
	Max  int // 最大存储字节数
	sync.Mutex
}

// TerminalSession 终端会话
type TerminalSession struct {
	ID string // 会话ID
	Cmd          *exec.Cmd      // 命令进程
	Ptmx         *os.File       // PTY主设备
	Done         chan struct{}  // 完成信号
	Stdin        io.WriteCloser // 标准输入（用于非PTY模式）
	IsStandardPipe bool         // 是否使用标准管道（非PTY模式）
	LastActive   time.Time      // 上次活跃时间
	Mutex        sync.Mutex     // 锁，确保线程安全
	
	// 新增字段
	Clients      map[string]*websocket.Conn // 连接到该会话的客户端
	ClientsMutex sync.Mutex                 // 客户端列表的互斥锁
	Buffer       OutputBuffer               // 输出缓冲区，用于新客户端连接时回放
	Active       bool                       // 会话是否活跃
	Created      time.Time                  // 会话创建时间
	CancelFunc   context.CancelFunc         // 用于取消goroutine的函数
}

// 全局终端会话管理
var (
	// 用户名 -> 终端ID -> 会话
	TerminalSessions    = make(map[string]map[string]*TerminalSession)
	TerminalSessionsMux sync.Mutex
)

// Append 添加数据到缓冲区
func (b *OutputBuffer) Append(data []byte) {
	b.Lock()
	defer b.Unlock()
	
	// 计算新数据后的总大小
	newSize := len(b.Data) + len(data)
	
	// 如果超过最大大小，截断旧数据
	if newSize > b.Max {
		// 保留最后的数据
		excess := newSize - b.Max
		if excess < len(b.Data) {
			b.Data = b.Data[excess:]
		} else {
			b.Data = []byte{}
		}
	}
	
	// 添加新数据
	b.Data = append(b.Data, data...)
} 
