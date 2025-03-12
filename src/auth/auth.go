package auth

import (
	"log"	
	"ghosteye/models"
	"ghosteye/utils"
)

// SaveSessionToken 保存会话token
func SaveSessionToken(username string) string {
	token := utils.GenerateToken()
	
	models.SessionTokensMux.Lock()
	models.SessionTokens[token] = username
	models.SessionTokensMux.Unlock()
	
	log.Printf("Created new session token for user %s", username)
	
	return token
}

// ValidateToken 验证会话token
func ValidateToken(token string) (string, bool) {
	models.SessionTokensMux.Lock()
	defer models.SessionTokensMux.Unlock()
	
	username, exists := models.SessionTokens[token]
	return username, exists
}

// ValidateSessionToken 是 ValidateToken 的别名，保持代码兼容性
func ValidateSessionToken(token string) (string, bool) {
	return ValidateToken(token)
}

// RemoveToken 删除会话token
func RemoveToken(token string) {
	models.SessionTokensMux.Lock()
	defer models.SessionTokensMux.Unlock()
	
	delete(models.SessionTokens, token)
}

// GetUserState 获取用户状态
func GetUserState(username string) *models.UserState {
	models.UserStatesMux.Lock()
	defer models.UserStatesMux.Unlock()
	
	userState, exists := models.UserStates[username]
	if !exists {
		userState = &models.UserState{
			IsRunning: false,
			Command:   "",
		}
		models.UserStates[username] = userState
	}
	
	return userState
} 