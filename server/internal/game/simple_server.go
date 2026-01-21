package game

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	
	"github.com/czh0526/game/server/internal/did"
	"github.com/czh0526/game/server/internal/vc"
)

// SimpleServer 简化的游戏服务器
type SimpleServer struct {
	didService *did.SimpleService
	vcService  *vc.SimpleService
	upgrader   websocket.Upgrader
	
	// 游戏状态管理
	rooms     map[string]*GameRoom
	players   map[string]*Player
	roomMutex sync.RWMutex
}

// NewSimpleServer 创建新的简化游戏服务器
func NewSimpleServer(didService *did.SimpleService, vcService *vc.SimpleService) (*SimpleServer, error) {
	return &SimpleServer{
		didService: didService,
		vcService:  vcService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境需要更严格的检查
			},
		},
		rooms:   make(map[string]*GameRoom),
		players: make(map[string]*Player),
	}, nil
}

// HandleWebSocket 处理WebSocket连接
func (s *SimpleServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WebSocket connection from %s", r.RemoteAddr)

	// 处理连接
	s.handleConnection(conn)
}

// handleConnection 处理WebSocket连接
func (s *SimpleServer) handleConnection(conn *websocket.Conn) {
	var player *Player
	
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read message error: %v", err)
			break
		}

		msg.Timestamp = time.Now()

		// 处理消息
		switch msg.Type {
		case MsgTypeAuth:
			player = s.handleAuth(conn, &msg)
		case MsgTypeJoinRoom:
			if player != nil {
				s.handleJoinRoom(player, &msg)
			}
		case MsgTypeLeaveRoom:
			if player != nil {
				s.handleLeaveRoom(player, &msg)
			}
		case MsgTypePlayerMove:
			if player != nil {
				s.handlePlayerMove(player, &msg)
			}
		case MsgTypePlayerAction:
			if player != nil {
				s.handlePlayerAction(player, &msg)
			}
		case MsgTypeChat:
			if player != nil {
				s.handleChat(player, &msg)
			}
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}

	// 连接断开时清理
	if player != nil {
		s.handleDisconnect(player)
	}
}

// handleAuth 处理身份认证
func (s *SimpleServer) handleAuth(conn *websocket.Conn, msg *Message) *Player {
	authData, ok := msg.Data.(map[string]interface{})
	if !ok {
		s.sendError(conn, "Invalid auth data")
		return nil
	}

	playerDID, ok := authData["did"].(string)
	if !ok {
		s.sendError(conn, "Missing player DID")
		return nil
	}

	// 验证DID
	didResponse, err := s.didService.ResolveDID(playerDID)
	if err != nil {
		s.sendError(conn, fmt.Sprintf("Invalid DID: %v", err))
		return nil
	}

	// 创建或获取玩家
	player := s.getOrCreatePlayer(playerDID, didResponse.DIDDoc.ID)
	player.Connection = conn
	player.Status = "online"
	player.LastSeen = time.Now()

	// 发送认证成功消息
	authResponse := Message{
		Type: MsgTypeAuth,
		Data: map[string]interface{}{
			"success":  true,
			"playerId": player.ID,
			"did":      player.DID,
			"nickname": player.Nickname,
		},
		Timestamp: time.Now(),
	}
	conn.WriteJSON(authResponse)

	log.Printf("Player authenticated: %s (%s)", player.Nickname, player.DID)
	return player
}

// handleCompleteTask 处理完成任务
func (s *SimpleServer) handleCompleteTask(player *Player, actionData map[string]interface{}) {
	taskID, ok := actionData["taskId"].(string)
	if !ok {
		return
	}

	// 查找任务
	var task *Task
	for _, t := range player.Room.GameState.Tasks {
		if t.ID == taskID {
			task = t
			break
		}
	}

	if task == nil || task.Status != "active" {
		return
	}

	// 标记任务完成
	task.Status = "completed"

	// 颁发成就凭证
	credential, err := s.vcService.IssueAchievementCredential(
		player.DID, 
		player.Room.GameID, 
		player.ID, 
		task.Name, 
		100, // 默认分数
	)
	if err != nil {
		log.Printf("Failed to issue credential: %v", err)
		return
	}

	// 通知玩家获得凭证
	player.Connection.WriteJSON(Message{
		Type:     MsgTypeCredential,
		PlayerID: player.ID,
		Data: map[string]interface{}{
			"credential": credential,
			"message":    fmt.Sprintf("获得凭证: %s", task.Name),
		},
		Timestamp: time.Now(),
	})

	// 通知任务完成
	s.broadcastToRoom(player.Room, Message{
		Type:     MsgTypeTaskUpdate,
		PlayerID: player.ID,
		RoomID:   player.Room.ID,
		Data: map[string]interface{}{
			"task":   task,
			"action": "completed",
		},
		Timestamp: time.Now(),
	}, "")

	log.Printf("Player %s completed task %s", player.Nickname, task.Name)
}

// 其他方法保持不变，只是简化了依赖
func (s *SimpleServer) getOrCreatePlayer(playerDID, didID string) *Player {
	s.roomMutex.Lock()
	defer s.roomMutex.Unlock()

	// 尝试从现有玩家中查找
	for _, player := range s.players {
		if player.DID == playerDID {
			return player
		}
	}

	// 创建新玩家
	player := &Player{
		ID:        uuid.New().String(),
		DID:       playerDID,
		Nickname:  fmt.Sprintf("Player_%s", playerDID[len(playerDID)-8:]),
		Position:  Position{X: 0, Y: 0},
		Level:     1,
		Health:    100,
		MaxHealth: 100,
		Status:    "online",
		LastSeen:  time.Now(),
	}

	s.players[player.ID] = player
	return player
}

func (s *SimpleServer) handleJoinRoom(player *Player, msg *Message) {
	joinData, ok := msg.Data.(map[string]interface{})
	if !ok {
		s.sendErrorToPlayer(player, "Invalid join room data")
		return
	}

	roomID, ok := joinData["roomId"].(string)
	if !ok {
		roomID = "default"
	}

	room := s.getOrCreateRoom(roomID, "default")
	
	if err := s.joinRoom(player, room); err != nil {
		s.sendErrorToPlayer(player, fmt.Sprintf("Failed to join room: %v", err))
		return
	}

	joinResponse := Message{
		Type:     MsgTypeJoinRoom,
		PlayerID: player.ID,
		RoomID:   room.ID,
		Data: map[string]interface{}{
			"success":   true,
			"room":      room,
			"gameState": room.GameState,
		},
		Timestamp: time.Now(),
	}
	player.Connection.WriteJSON(joinResponse)

	s.broadcastToRoom(room, Message{
		Type:     MsgTypePlayerUpdate,
		PlayerID: player.ID,
		RoomID:   room.ID,
		Data: map[string]interface{}{
			"action": "joined",
			"player": player,
		},
		Timestamp: time.Now(),
	}, player.ID)

	log.Printf("Player %s joined room %s", player.Nickname, room.ID)
}

func (s *SimpleServer) getOrCreateRoom(roomID, gameID string) *GameRoom {
	s.roomMutex.Lock()
	defer s.roomMutex.Unlock()

	room, exists := s.rooms[roomID]
	if exists {
		return room
	}

	room = &GameRoom{
		ID:         roomID,
		Name:       fmt.Sprintf("Room %s", roomID),
		GameID:     gameID,
		MaxPlayers: 10,
		Players:    make(map[string]*Player),
		GameState:  s.createDefaultGameState(),
		CreatedAt:  time.Now(),
	}

	s.rooms[roomID] = room
	log.Printf("Created new room: %s", roomID)
	return room
}

func (s *SimpleServer) createDefaultGameState() *GameState {
	return &GameState{
		Status: "waiting",
		Map: &GameMap{
			Width:  800,
			Height: 600,
			Tiles:  make([][]int, 60),
			Objects: []*MapObject{
				{
					ID:   "spawn1",
					Type: "spawn_point",
					Position: Position{X: 100, Y: 100},
					Properties: map[string]interface{}{
						"team": "default",
					},
				},
			},
			SpawnPoints: []Position{
				{X: 100, Y: 100},
				{X: 200, Y: 200},
				{X: 300, Y: 300},
			},
		},
		Tasks: []*Task{
			{
				ID:          "welcome_task",
				Name:        "Welcome to the Game",
				Description: "Complete your first steps in the game",
				Type:        "tutorial",
				Status:      "available",
				Objectives: []*Objective{
					{
						ID:          "move_around",
						Description: "Move using WASD keys",
						Type:        "movement",
						Target:      "any",
						Current:     0,
						Required:    1,
						Completed:   false,
					},
				},
				Rewards: []*Reward{
					{
						Type:  "credential",
						Value: "WelcomeCredential",
					},
				},
			},
		},
		Events:     []*GameEvent{},
		Properties: make(map[string]interface{}),
	}
}

func (s *SimpleServer) joinRoom(player *Player, room *GameRoom) error {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	if len(room.Players) >= room.MaxPlayers {
		return fmt.Errorf("room is full")
	}

	if player.Room != nil {
		s.leaveRoom(player)
	}

	room.Players[player.ID] = player
	player.Room = room

	if len(room.GameState.Map.SpawnPoints) > 0 {
		spawnIndex := len(room.Players) % len(room.GameState.Map.SpawnPoints)
		player.Position = room.GameState.Map.SpawnPoints[spawnIndex]
	}

	return nil
}

func (s *SimpleServer) handleLeaveRoom(player *Player, msg *Message) {
	if player.Room == nil {
		return
	}

	room := player.Room
	s.leaveRoom(player)

	leaveResponse := Message{
		Type:     MsgTypeLeaveRoom,
		PlayerID: player.ID,
		Data: map[string]interface{}{
			"success": true,
		},
		Timestamp: time.Now(),
	}
	player.Connection.WriteJSON(leaveResponse)

	s.broadcastToRoom(room, Message{
		Type:     MsgTypePlayerUpdate,
		PlayerID: player.ID,
		RoomID:   room.ID,
		Data: map[string]interface{}{
			"action": "left",
			"player": player,
		},
		Timestamp: time.Now(),
	}, player.ID)
}

func (s *SimpleServer) leaveRoom(player *Player) {
	if player.Room == nil {
		return
	}

	room := player.Room
	room.mutex.Lock()
	defer room.mutex.Unlock()

	delete(room.Players, player.ID)
	player.Room = nil

	if len(room.Players) == 0 {
		s.roomMutex.Lock()
		delete(s.rooms, room.ID)
		s.roomMutex.Unlock()
		log.Printf("Deleted empty room: %s", room.ID)
	}
}

func (s *SimpleServer) handlePlayerMove(player *Player, msg *Message) {
	if player.Room == nil {
		return
	}

	moveData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	x, xOk := moveData["x"].(float64)
	y, yOk := moveData["y"].(float64)
	if !xOk || !yOk {
		return
	}

	player.Position.X = x
	player.Position.Y = y

	s.broadcastToRoom(player.Room, Message{
		Type:     MsgTypePlayerMove,
		PlayerID: player.ID,
		RoomID:   player.Room.ID,
		Data: map[string]interface{}{
			"position": player.Position,
		},
		Timestamp: time.Now(),
	}, player.ID)
}

func (s *SimpleServer) handlePlayerAction(player *Player, msg *Message) {
	if player.Room == nil {
		return
	}

	actionData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	action, ok := actionData["action"].(string)
	if !ok {
		return
	}

	switch action {
	case "complete_task":
		s.handleCompleteTask(player, actionData)
	case "interact":
		s.handleInteract(player, actionData)
	default:
		log.Printf("Unknown action: %s", action)
	}
}

func (s *SimpleServer) handleInteract(player *Player, actionData map[string]interface{}) {
	// 简化的交互处理
	log.Printf("Player %s interacted", player.Nickname)
}

func (s *SimpleServer) handleChat(player *Player, msg *Message) {
	if player.Room == nil {
		return
	}

	chatData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	message, ok := chatData["message"].(string)
	if !ok {
		return
	}

	s.broadcastToRoom(player.Room, Message{
		Type:     MsgTypeChat,
		PlayerID: player.ID,
		RoomID:   player.Room.ID,
		Data: map[string]interface{}{
			"message":  message,
			"nickname": player.Nickname,
		},
		Timestamp: time.Now(),
	}, "")
}

func (s *SimpleServer) handleDisconnect(player *Player) {
	player.Status = "offline"
	player.Connection = nil

	if player.Room != nil {
		s.broadcastToRoom(player.Room, Message{
			Type:     MsgTypePlayerUpdate,
			PlayerID: player.ID,
			RoomID:   player.Room.ID,
			Data: map[string]interface{}{
				"action": "disconnected",
				"player": player,
			},
			Timestamp: time.Now(),
		}, player.ID)
	}

	log.Printf("Player %s disconnected", player.Nickname)
}

func (s *SimpleServer) broadcastToRoom(room *GameRoom, msg Message, excludePlayerID string) {
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	for playerID, player := range room.Players {
		if playerID != excludePlayerID && player.Connection != nil {
			player.Connection.WriteJSON(msg)
		}
	}
}

func (s *SimpleServer) sendError(conn *websocket.Conn, message string) {
	errorMsg := Message{
		Type: MsgTypeError,
		Data: map[string]interface{}{
			"message": message,
		},
		Timestamp: time.Now(),
	}
	conn.WriteJSON(errorMsg)
}

func (s *SimpleServer) sendErrorToPlayer(player *Player, message string) {
	if player.Connection != nil {
		s.sendError(player.Connection, message)
	}
}