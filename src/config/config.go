package config

// Config 存储应用程序的配置
type Config struct {
	ServerPort   string
	Username     string
	Password     string
	RandomUsers  int
	WhitelistIPs string
	ShowUsers    bool
}

// 全局配置实例
var AppConfig Config

// Initialize 初始化应用程序配置
func Initialize(serverPort, username, password string, randomUsers int, whitelistIPs string, showUsers bool) {
	AppConfig = Config{
		ServerPort:   serverPort,
		Username:     username,
		Password:     password,
		RandomUsers:  randomUsers,
		WhitelistIPs: whitelistIPs,
		ShowUsers:    showUsers,
	}
}

// GetServerPort 获取服务器端口
func GetServerPort() string {
	return AppConfig.ServerPort
}

// GetUsername 获取管理员用户名
func GetUsername() string {
	return AppConfig.Username
}

// GetPassword 获取管理员密码
func GetPassword() string {
	return AppConfig.Password
}

// GetRandomUsers 获取随机用户数量
func GetRandomUsers() int {
	return AppConfig.RandomUsers
}

// GetWhitelistIPs 获取白名单IP地址
func GetWhitelistIPs() string {
	return AppConfig.WhitelistIPs
}

// ShouldShowUsers 是否显示所有用户
func ShouldShowUsers() bool {
	return AppConfig.ShowUsers
} 