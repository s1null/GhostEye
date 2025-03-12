package api

import (
	"encoding/json"
	"log"
	"net/http"
	
	"ghosteye/auth"
	"ghosteye/database"
	"ghosteye/middleware"
	"ghosteye/models"
	"ghosteye/utils"
	"ghosteye/terminal"
)

// StartHandler 用于接收启动指令
func StartHandler(w http.ResponseWriter, r *http.Request) {
	userState := middleware.GetUserState(r)
	if userState == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get user state"))
		return
	}

	userState.Mutex.Lock()
	defer userState.Mutex.Unlock()

	// 获取命令参数
	cmd := r.URL.Query().Get("cmd")
	if cmd == "" {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Missing parameter: cmd"})
		return
	}

	// 更新用户状态
	userState.IsRunning = true
	userState.Command = cmd

	// 返回成功响应
	utils.WriteJSON(w, models.Response{Code: 0, Message: "Command received", Data: map[string]string{"cmd": cmd}})
}

// StopHandler 用于接收停止指令
func StopHandler(w http.ResponseWriter, r *http.Request) {
	userState := middleware.GetUserState(r)
	if userState == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get user state"))
		return
	}

	userState.Mutex.Lock()
	defer userState.Mutex.Unlock()

	// 更新用户状态
	userState.IsRunning = false
	userState.Command = ""

	// 返回成功响应
	utils.WriteJSON(w, models.Response{Code: 0, Message: "Command stopped"})
}

// StatusHandler 用于查询命令状态
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	userState := middleware.GetUserState(r)
	if userState == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get user state"))
		return
	}

	userState.Mutex.Lock()
	defer userState.Mutex.Unlock()

	// 返回状态
	utils.WriteJSON(w, models.Response{
		Code:    0,
		Message: "Status retrieved",
		Data: map[string]interface{}{
			"is_running": userState.IsRunning,
			"command":    userState.Command,
		},
	})
}

// LoginHandler 处理用户登录
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// 只接受POST请求
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}

	// 解析请求体
	var authReq models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Invalid request format"})
		return
	}

	// 验证用户名和密码
	if !database.ValidateUser(authReq.Username, authReq.Password) {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Invalid username or password"})
		return
	}

	// 生成会话令牌
	token := auth.SaveSessionToken(authReq.Username)

	// 返回成功响应
	utils.WriteJSON(w, models.Response{
		Code:    0,
		Message: "Login successful",
		Data: map[string]string{
			"token":    token,
			"username": authReq.Username,
		},
	})
}

// IndexHandler 处理根路径请求
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// 如果请求的是根路径，重定向到web目录
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "web/index.html")
		return
	}

	// 否则尝试提供静态文件
	http.ServeFile(w, r, "web"+r.URL.Path)
}

// GetUserCommandsHandler 获取用户命令
func GetUserCommandsHandler(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsernameFromContext(r)
	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}

	commands, err := database.GetUserCommands(username)
	if err != nil {
		log.Printf("Failed to get user commands: %v", err)
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Failed to get user commands"})
		return
	}

	utils.WriteJSON(w, models.Response{
		Code:    0,
		Message: "Commands retrieved",
		Data:    commands,
	})
}

// AddUserCommandHandler 添加用户命令
func AddUserCommandHandler(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsernameFromContext(r)
	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}

	// 解析请求体
	var cmd struct {
		Name        string `json:"name"`
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Invalid request format"})
		return
	}

	// 验证参数
	if cmd.Name == "" || cmd.Command == "" {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Name and command are required"})
		return
	}

	// 添加命令
	err := database.AddUserCommand(username, cmd.Name, cmd.Command, cmd.Description)
	if err != nil {
		log.Printf("Failed to add user command: %v", err)
		utils.WriteJSON(w, models.Response{Code: 1, Message: err.Error()})
		return
	}

	utils.WriteJSON(w, models.Response{
		Code:    0,
		Message: "Command added",
	})
}

// UpdateUserCommandHandler 更新用户命令
func UpdateUserCommandHandler(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsernameFromContext(r)
	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}

	// 解析请求体
	var cmd struct {
		Name        string `json:"name"`
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Invalid request format"})
		return
	}

	// 验证参数
	if cmd.Name == "" || cmd.Command == "" {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Name and command are required"})
		return
	}

	// 更新命令
	err := database.UpdateUserCommand(username, cmd.Name, cmd.Command, cmd.Description)
	if err != nil {
		log.Printf("Failed to update user command: %v", err)
		utils.WriteJSON(w, models.Response{Code: 1, Message: err.Error()})
		return
	}

	utils.WriteJSON(w, models.Response{
		Code:    0,
		Message: "Command updated",
	})
}

// DeleteUserCommandHandler 删除用户命令
func DeleteUserCommandHandler(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsernameFromContext(r)
	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}

	// 获取命令名称
	name := r.URL.Query().Get("name")
	if name == "" {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Missing parameter: name"})
		return
	}

	// 删除命令
	err := database.DeleteUserCommand(username, name)
	if err != nil {
		log.Printf("Failed to delete user command: %v", err)
		utils.WriteJSON(w, models.Response{Code: 1, Message: err.Error()})
		return
	}

	utils.WriteJSON(w, models.Response{
		Code:    0,
		Message: "Command deleted",
	})
}

// ListTerminalSessionsHandler 列出终端会话
func ListTerminalSessionsHandler(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsernameFromContext(r)
	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}

	sessions, err := database.GetUserTerminalSessions(username)
	if err != nil {
		log.Printf("Failed to get user terminal sessions: %v", err)
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Failed to get terminal sessions"})
		return
	}

	utils.WriteJSON(w, models.Response{
		Code:    0,
		Message: "Terminal sessions retrieved",
		Data:    sessions,
	})
}

// KillTerminalHandler 处理终止终端的请求
func KillTerminalHandler(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsernameFromContext(r)
	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}

	// 获取终端ID
	terminalID := r.URL.Query().Get("terminal_id")
	if terminalID == "" {
		utils.WriteJSON(w, models.Response{Code: 1, Message: "Missing parameter: terminal_id"})
		return
	}

	// 终止终端会话
	err := terminal.KillTerminalSession(username, terminalID)
	if err != nil {
		log.Printf("Failed to terminate terminal session: %v", err)
		utils.WriteJSON(w, models.Response{Code: 1, Message: err.Error()})
		return
	}

	utils.WriteJSON(w, models.Response{
		Code:    0,
		Message: "Terminal session terminated",
	})
} 
