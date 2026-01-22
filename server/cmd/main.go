package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/czh0526/game/server/internal/aries"
	"github.com/czh0526/game/server/internal/game"
	"github.com/czh0526/game/server/internal/did"
	"github.com/czh0526/game/server/internal/vc"
)

func main() {
	var (
		addr = flag.String("addr", ":8080", "HTTP server address")
		staticDir = flag.String("static", "./client", "Static files directory")
		mysqlDSN = flag.String("mysql-dsn", "root:password@tcp(localhost:3308)/aries_did?parseTime=true", "MySQL data source name")
	)
	flag.Parse()

	// 初始化Aries服务
	log.Println("Initializing Aries service with MySQL storage...")
	ariesSvc, err := aries.NewAriesService(&aries.Config{
		MySQLDSN: *mysqlDSN,
		Label:    "game-did-service",
	})
	if err != nil {
		log.Fatalf("Failed to initialize Aries service: %v", err)
	}
	defer ariesSvc.Close()
	log.Println("Aries service initialized successfully")

	// 初始化DID服务（使用Aries）
	didService := did.NewSimpleServiceWithAries(ariesSvc)

	// 初始化VC服务
	vcService, err := vc.NewSimpleService(didService)
	if err != nil {
		log.Fatalf("Failed to initialize VC service: %v", err)
	}

	// 初始化游戏服务器
	gameServer, err := game.NewSimpleServer(didService, vcService)
	if err != nil {
		log.Fatalf("Failed to initialize game server: %v", err)
	}

	// 设置HTTP路由
	mux := http.NewServeMux()

	// 静态文件服务
	mux.Handle("/", http.FileServer(http.Dir(*staticDir)))

	// API路由 - DID管理
	mux.HandleFunc("/api/did/create", didService.HandleCreateDIDWithAries)
	mux.HandleFunc("/api/did/register", didService.HandleRegisterDID)
	mux.HandleFunc("/api/did/resolve", didService.HandleResolveDID)

	// API路由 - VC管理
	mux.HandleFunc("/api/vc/issue", vcService.HandleIssueCredential)
	mux.HandleFunc("/api/vc/verify", vcService.HandleVerifyCredential)

	// WebSocket游戏连接
	mux.HandleFunc("/ws/game", gameServer.HandleWebSocket)

	server := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	// 启动服务器
	go func() {
		log.Printf("Starting server on %s", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}