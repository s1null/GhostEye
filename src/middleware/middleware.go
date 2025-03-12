package middleware

import (
	"context"
	"net/http"
	"strings"
	
	"ghosteye/auth"
	"ghosteye/database"
	"ghosteye/models"
	"ghosteye/utils"
)

// 上下文键类型
type contextKey string

// 用户名上下文键
const usernameKey contextKey = "username"

// ContextWithUsername 将用户名添加到上下文
func ContextWithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey, username)
}

// UsernameFromContext 从上下文获取用户名
func UsernameFromContext(ctx context.Context) string {
	if username, ok := ctx.Value(usernameKey).(string); ok {
		return username
	}
	return ""
}

// CorsMiddleware 添加CORS支持
func CorsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// 调用下一个处理函数
		next(w, r)
	}
}

// IPWhitelistMiddleware 检查请求IP是否在白名单中
func IPWhitelistMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取客户端IP
		ip := utils.GetClientIP(r)
		
		// 检查IP是否在白名单中
		if !database.IsIPInWhitelist(ip) {
			// 使用http.Hijacker劫持连接并关闭，模拟端口未开放的行为
			hj, ok := w.(http.Hijacker)
			if ok {
				// 获取底层连接并关闭
				conn, _, err := hj.Hijack()
				if err == nil {
					conn.Close() // 直接关闭连接，不发送任何响应
					return
				}
			}
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		
		// 调用下一个处理函数
		next(w, r)
	}
}

// TokenAuth 通过会话令牌进行认证
func TokenAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// 获取Authorization头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Authentication token not provided"))
			return
		}

		// 解析Authorization头
		authParts := strings.SplitN(authHeader, " ", 2)
		if len(authParts) != 2 || authParts[0] != "Bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid authentication format, should be Bearer token"))
			return
		}

		// 获取令牌
		token := authParts[1]
		
		// 验证令牌
		username, valid := auth.ValidateSessionToken(token)
		if !valid {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid or expired token"))
			return
		}

		// 获取或创建用户状态
		auth.GetUserState(username)

		// 将用户名存储在请求上下文中
		r = SetUsernameInContext(r, username)

		// 调用下一个处理函数
		next(w, r)
	}
}

// SetUsernameInContext 在请求上下文中设置用户名
func SetUsernameInContext(r *http.Request, username string) *http.Request {
	ctx := r.Context()
	ctx = ContextWithUsername(ctx, username)
	return r.WithContext(ctx)
}

// GetUsernameFromContext 从请求上下文中获取用户名
func GetUsernameFromContext(r *http.Request) string {
	return UsernameFromContext(r.Context())
}

// GetUserState 从请求中获取用户状态
func GetUserState(r *http.Request) *models.UserState {
	username := GetUsernameFromContext(r)
	if username == "" {
		return nil
	}

	return auth.GetUserState(username)
} 
