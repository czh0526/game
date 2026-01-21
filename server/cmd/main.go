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

	"github.com/czh0526/game/server/internal/game"
	"github.com/czh0526/game/server/internal/did"
	"github.com/czh0526/game/server/internal/vc"
)

func main() {
	var (
		addr = flag.String("addr", ":8080", "HTTP server address")
		staticDir = flag.String("static", "./client", "Static files directory")
	)
	flag.Parse()

	// 初始化DID服务
	didService := did.NewSimpleService()

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
	
	// API路由
	mux.HandleFunc("/api/did/create", didService.HandleCreateDID)
	mux.HandleFunc("/api/did/resolve", didService.HandleResolveDID)
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