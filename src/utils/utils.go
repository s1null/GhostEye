package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	
	"ghosteye/models"
)

// WriteJSON 用于将响应写入HTTP响应中
func WriteJSON(w http.ResponseWriter, resp models.Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GenerateSessionID 生成随机session id
func GenerateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// GenerateToken 生成会话token
func GenerateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// GetClientIP 获取客户端真实IP地址
func GetClientIP(r *http.Request) string {
	// 直接从 RemoteAddr 获取，这是 TCP 连接的实际来源 IP
	// RemoteAddr 格式为"IP:端口"，需要提取 IP 部分
	remoteAddr := r.RemoteAddr
	if remoteAddr == "" {
		return ""
	}
	
	// 提取 IP 部分
	if strings.Contains(remoteAddr, ":") {
		return strings.Split(remoteAddr, ":")[0]
	}
	
	return remoteAddr
} 