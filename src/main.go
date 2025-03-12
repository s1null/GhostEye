package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	
	"ghosteye/api"
	"ghosteye/config"
	"ghosteye/database"
	"ghosteye/middleware"
	"ghosteye/terminal"
)

//go:embed web/out/*
var webContent embed.FS

func main() {
	// 解析命令行参数
	serverPort := flag.String("p", "8080", "Server listening port")
	username := flag.String("user", "", "Specify admin username")
	password := flag.String("pass", "", "Specify admin password")
	randomUsers := flag.Int("U", 0, "Auto-generate specified number of random users")
	whitelistIPs := flag.String("w", "", "Whitelist IP addresses, separate multiple IPs with commas")
	showUsers := flag.Bool("show-users", false, "Show all user account information")
	flag.Parse()

	// 初始化配置
	config.Initialize(*serverPort, *username, *password, *randomUsers, *whitelistIPs, *showUsers)

	// 初始化数据库
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.CloseDatabase()

	// 处理白名单IP
	if *whitelistIPs != "" {
		ips := strings.Split(*whitelistIPs, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				if err := database.AddIPToWhitelist(ip, "Added through command line parameters"); err != nil {
					log.Printf("Failed to add IP to whitelist: %v", err)
				}
			}
		}
	}

	// 处理用户账号
	if *username != "" && *password != "" {
		// 添加指定的管理员账号
		if err := database.AddUser(*username, *password); err != nil {
			log.Printf("Failed to add admin account: %v", err)
		} else {
			log.Printf("Admin account %s has been added", *username)
		}
	}

	// 生成随机用户
	if *randomUsers > 0 {
		users := database.GenerateRandomUsers(*randomUsers)
		log.Printf("Generated %d random users:", len(users))
		for _, user := range users {
			log.Printf("Username: %s, Password: %s", user["username"], user["password"])
		}
	}

	// 显示所有用户
	if *showUsers {
		users, err := database.GetAllUsers()
		if err != nil {
			log.Printf("Failed to get user list: %v", err)
		} else {
			log.Printf("There are %d users in the system:", len(users))
			for _, user := range users {
				log.Printf("Username: %s, Password: %s", user["username"], user["password"])
			}
		}
	}

	// 如果没有任何用户，添加默认管理员账号
	users, _ := database.GetAllUsers()
	if len(users) == 0 {
		defaultUsername := "admin"
		defaultPassword := "admin"
		if err := database.AddUser(defaultUsername, defaultPassword); err != nil {
			log.Printf("Failed to add default admin account: %v", err)
		} else {
			log.Printf("Default admin account added - Username: %s, Password: %s", defaultUsername, defaultPassword)
		}
	}

	// 初始化HTTP路由
	mux := http.NewServeMux()
	
	// 设置WebSocket处理
	mux.HandleFunc("/ws", middleware.IPWhitelistMiddleware(terminal.TerminalHandler))
	
	// 设置API路由
	mux.HandleFunc("/api/login", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(api.LoginHandler)))
	mux.HandleFunc("/api/status", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.StatusHandler))))
	mux.HandleFunc("/api/start", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.StartHandler))))
	mux.HandleFunc("/api/stop", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.StopHandler))))
	
	// 终端会话相关API
	mux.HandleFunc("/api/terminals", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.ListTerminalSessionsHandler))))
	mux.HandleFunc("/api/terminals/kill", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.KillTerminalHandler))))
	
	// 命令相关API
	mux.HandleFunc("/api/commands", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.GetUserCommandsHandler))))
	mux.HandleFunc("/api/commands/add", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.AddUserCommandHandler))))
	mux.HandleFunc("/api/commands/update", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.UpdateUserCommandHandler))))
	mux.HandleFunc("/api/commands/delete", middleware.IPWhitelistMiddleware(middleware.CorsMiddleware(middleware.TokenAuth(api.DeleteUserCommandHandler))))
	
	// 设置静态文件服务
	// 初始化嵌入式前端资源
	log.Println("Using embedded frontend resources")
	webFS, err := fs.Sub(webContent, "web/out")
	if err != nil {
		log.Fatalf("Failed to access embedded frontend resources: %v", err)
	}
	fileServer := http.FileServer(http.FS(webFS))
	
	// 设置前端路由处理
	mux.HandleFunc("/", middleware.IPWhitelistMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径是否是API路径或WS路径
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}
		
		// 检查路径是否是静态资源（CSS, JS等）
		if isStaticResource(r.URL.Path) {
			fileServer.ServeHTTP(w, r)
			return
		}
		
		// 对于所有其他路径，包括根路径和客户端路由路径，返回index.html
		content, err := webContent.ReadFile("web/out/index.html")
		if err != nil {
			log.Printf("Failed to read embedded index.html: %v\n", err)
			http.Error(w, "Frontend file not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	}))
	
	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":" + config.GetServerPort(),
		Handler: mux,
	}
	
	// 启动终端会话清理定时器
	terminal.StartSessionCleaner()
	
	// 启动服务器
	log.Printf("Server started, listening on port: %s\n", config.GetServerPort())
	
	// 在单独的goroutine中启动服务器，这样它就不会阻塞优雅退出的处理
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listening failed: %v", err)
		}
	}()
	
	// 设置优雅退出
	gracefulShutdown()
}

// 判断是否是静态资源
func isStaticResource(path string) bool {
	// 常见的静态文件后缀
	staticExtensions := []string{
		".js", ".css", ".html", ".png", ".jpg", ".jpeg", ".gif",
		".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".json",
		".map", ".txt", ".pdf", ".webp",
	}
	
	// 特定的静态目录前缀
	staticPrefixes := []string{
		"/_next/", "/images/", "/assets/", "/static/", "/favicon.ico",
	}
	
	// 检查前缀
	for _, prefix := range staticPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	
	// 检查后缀
	for _, ext := range staticExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	
	return false
}

// 设置优雅关闭
func gracefulShutdown() {
	// 监听 Ctrl+C 和 kill 命令
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	
	// 阻塞直到收到终止信号
	<-stop
	
	log.Println("Shutting down server...")
	
	// 这里可以添加任何优雅关闭逻辑，如关闭数据库连接等
	database.CloseDatabase()
	
	log.Println("Server has been safely shut down, goodbye!")
}
