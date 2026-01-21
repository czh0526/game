package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WebSocket connection from %s", r.RemoteAddr)

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read message error: %v", err)
			break
		}

		log.Printf("Received message: %s", msg.Type)

		// Echo back the message
		response := Message{
			Type: "response",
			Data: map[string]interface{}{
				"received": msg.Type,
				"message":  "Hello from Aries Game Server!",
			},
			Timestamp: time.Now(),
		}

		if err := conn.WriteJSON(response); err != nil {
			log.Printf("Write message error: %v", err)
			break
		}
	}
}

func handleCreateDID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"did":        "did:player:default:test123",
		"publicKey":  "abcd1234",
		"privateKey": "efgh5678",
		"didDocument": map[string]interface{}{
			"@context": []string{"https://www.w3.org/ns/did/v1"},
			"id":       "did:player:default:test123",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	mux := http.NewServeMux()

	// 静态文件服务
	mux.Handle("/", http.FileServer(http.Dir("./client")))

	// API路由
	mux.HandleFunc("/api/did/create", handleCreateDID)
	mux.HandleFunc("/ws/game", handleWebSocket)

	log.Println("Starting Aries Game Server on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
