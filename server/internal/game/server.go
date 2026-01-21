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

// Server 游戏服务器
type Server struct {
	didService *did.Service
	vcService  *vc.Service
	upgrader   websocket.Upgrader
	
	// 游戏状态管理
	rooms     map[string]*GameRoom
	players   map[string]*Player
	roomMutex sync.RWMutex
}

// Player 玩家信息
type Player struct {
	ID         string          `json:"id"`
	DID        string          `json:"did"`
	Nickname   string          `json:"nickname"`
	Position   Position        `json:"position"`
	Level      int             `json:"level"`
	Health     int             `json:"health"`
	MaxHealth  int             `json:"maxHealth"`
	Status     string          `json:"status"` // online, offline, playing
	Connection *websocket.Conn `json:"-"`
	Room       *GameRoom       `json:"-"`
	LastSeen   time.Time       `json:"lastSeen"`
}

// Position 位置信息
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GameRoom 游戏房间
type GameRoom struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	GameID      string             `json:"gameId"`
	MaxPlayers  int                `json:"maxPlayers"`
	Players     map[string]*Player `json:"players"`
	GameState   *GameState         `json:"gameState"`
	CreatedAt   time.Time          `json:"createdAt"`
	mutex       sync.RWMutex
}

// GameState 游戏状态
type GameState struct {
	Status      string                 `json:"status"` // waiting, playing, finished
	StartTime   *time.Time             `json:"startTime,omitempty"`
	EndTime     *time.Time             `json:"endTime,omitempty"`
	Map         *GameMap               `json:"map"`
	Tasks       []*Task                `json:"tasks"`
	Events      []*GameEvent           `json:"events"`
	Properties  map[string]interface{} `json:"properties"`
}

// GameMap 游戏地图
type GameMap struct {
	Width     int        `json:"width"`
	Height    int        `json:"height"`
	Tiles     [][]int    `json:"tiles"`
	Objects   []*MapObject `json:"objects"`
	SpawnPoints []Position `json:"spawnPoints"`
}

// MapObject 地图对象
type MapObject struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Position Position `json:"position"`
	Width    int      `json:"width"`
	Height   int      `json:"height"`
	Properties map[string]interface{} `json:"properties"`
}

// Task 游戏任务
type Task struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"` // available, active, completed, failed
	Objectives  []*Objective           `json:"objectives"`
	Rewards     []*Reward              `json:"rewards"`
	Properties  map[string]interface{} `json:"properties"`
}

// Objective 任务目标
type Objective struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Target      string                 `json:"target"`
	Current     int                    `json:"current"`
	Required    int                    `json:"required"`
	Completed   bool                   `json:"completed"`
	Properties  map[string]interface{} `json:"properties"`
}

// Reward 奖励
type Reward struct {
	Type       string                 `json:"type"`
	Value      interface{}            `json:"value"`
	Properties map[string]interface{} `json:"properties"`
}

// GameEvent 游戏事件
type GameEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	PlayerID  string                 `json:"playerId,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Message WebSocket消息
type Message struct {
	Type      string      `json:"type"`
	PlayerID  string      `json:"playerId,omitempty"`
	RoomID    string      `json:"roomId,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// 消息类型常量
const (
	MsgTypeJoinRoom      = "join_room"
	MsgTypeLeaveRoom     = "leave_room"
	MsgTypePlayerMove    = "player_move"
	MsgTypePlayerAction  = "player_action"
	MsgTypeTaskUpdate    = "task_update"
	MsgTypeGameState     = "game_state"
	MsgTypePlayerUpdate  = "player_update"
	MsgTypeChat          = "chat"
	MsgTypeError         = "error"
	MsgTypeAuth          = "auth"
	MsgTypeCredential    = "credential"
)

// NewServer 创建新的游戏服务器
func NewServer(didService *did.Service, vcService *vc.Service) (*Server, error) {
	return &Server{
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
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
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
func (s *Server) handleConnection(conn *websocket.Conn) {
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
func (s *Server) handleAuth(conn *websocket.Conn, msg *Message) *Player {
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

// getOrCreatePlayer 获取或创建玩家
func (s *Server) getOrCreatePlayer(playerDID, didID string) *Player {
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

// handleJoinRoom 处理加入房间
func (s *Server) handleJoinRoom(player *Player, msg *Message) {
	joinData, ok := msg.Data.(map[string]interface{})
	if !ok {
		s.sendErrorToPlayer(player, "Invalid join room data")
		return
	}

	roomID, ok := joinData["roomId"].(string)
	if !ok {
		// 如果没有指定房间，创建或加入默认房间
		roomID = "default"
	}

	room := s.getOrCreateRoom(roomID, "default")
	
	// 加入房间
	if err := s.joinRoom(player, room); err != nil {
		s.sendErrorToPlayer(player, fmt.Sprintf("Failed to join room: %v", err))
		return
	}

	// 通知玩家加入成功
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

	// 通知房间内其他玩家
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

// getOrCreateRoom 获取或创建房间
func (s *Server) getOrCreateRoom(roomID, gameID string) *GameRoom {
	s.roomMutex.Lock()
	defer s.roomMutex.Unlock()

	room, exists := s.rooms[roomID]
	if exists {
		return room
	}

	// 创建新房间
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

// createDefaultGameState 创建默认游戏状态
func (s *Server) createDefaultGameState() *GameState {
	return &GameState{
		Status: "waiting",
		Map: &GameMap{
			Width:  800,
			Height: 600,
			Tiles:  make([][]int, 60), // 简单的瓦片地图
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
		Tasks:      []*Task{},
		Events:     []*GameEvent{},
		Properties: make(map[string]interface{}),
	}
}

// joinRoom 玩家加入房间
func (s *Server) joinRoom(player *Player, room *GameRoom) error {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	if len(room.Players) >= room.MaxPlayers {
		return fmt.Errorf("room is full")
	}

	// 如果玩家已在其他房间，先离开
	if player.Room != nil {
		s.leaveRoom(player)
	}

	// 加入新房间
	room.Players[player.ID] = player
	player.Room = room

	// 设置玩家初始位置
	if len(room.GameState.Map.SpawnPoints) > 0 {
		spawnIndex := len(room.Players) % len(room.GameState.Map.SpawnPoints)
		player.Position = room.GameState.Map.SpawnPoints[spawnIndex]
	}

	return nil
}

// handleLeaveRoom 处理离开房间
func (s *Server) handleLeaveRoom(player *Player, msg *Message) {
	if player.Room == nil {
		return
	}

	room := player.Room
	s.leaveRoom(player)

	// 通知玩家离开成功
	leaveResponse := Message{
		Type:     MsgTypeLeaveRoom,
		PlayerID: player.ID,
		Data: map[string]interface{}{
			"success": true,
		},
		Timestamp: time.Now(),
	}
	player.Connection.WriteJSON(leaveResponse)

	// 通知房间内其他玩家
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

// leaveRoom 玩家离开房间
func (s *Server) leaveRoom(player *Player) {
	if player.Room == nil {
		return
	}

	room := player.Room
	room.mutex.Lock()
	defer room.mutex.Unlock()

	delete(room.Players, player.ID)
	player.Room = nil

	// 如果房间空了，可以考虑删除房间
	if len(room.Players) == 0 {
		s.roomMutex.Lock()
		delete(s.rooms, room.ID)
		s.roomMutex.Unlock()
		log.Printf("Deleted empty room: %s", room.ID)
	}
}

// handlePlayerMove 处理玩家移动
func (s *Server) handlePlayerMove(player *Player, msg *Message) {
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

	// 更新玩家位置
	player.Position.X = x
	player.Position.Y = y

	// 广播位置更新
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

// handlePlayerAction 处理玩家动作
func (s *Server) handlePlayerAction(player *Player, msg *Message) {
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

	// 处理不同类型的动作
	switch action {
	case "complete_task":
		s.handleCompleteTask(player, actionData)
	case "interact":
		s.handleInteract(player, actionData)
	default:
		log.Printf("Unknown action: %s", action)
	}
}

// handleCompleteTask 处理完成任务
func (s *Server) handleCompleteTask(player *Player, actionData map[string]interface{}) {
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

	// 颁发奖励和凭证
	for _, reward := range task.Rewards {
		s.processReward(player, reward, task)
	}

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

// processReward 处理奖励
func (s *Server) processReward(player *Player, reward *Reward, task *Task) {
	switch reward.Type {
	case "credential":
		// 颁发凭证
		credType := "AchievementCredential"
		if ct, ok := reward.Properties["credentialType"].(string); ok {
			credType = ct
		}

		subject := vc.GameCredentialSubject{
			PlayerID:    player.ID,
			GameID:      player.Room.GameID,
			Achievement: task.Name,
			Attributes: map[string]interface{}{
				"taskId":     task.ID,
				"difficulty": reward.Properties["difficulty"],
			},
		}

		credential, err := s.vcService.IssueCredential(player.DID, credType, subject, nil)
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

	case "experience":
		// 增加经验值
		if exp, ok := reward.Value.(float64); ok {
			// 这里可以实现经验值系统
			log.Printf("Player %s gained %d experience", player.Nickname, int(exp))
		}
	}
}

// handleChat 处理聊天消息
func (s *Server) handleChat(player *Player, msg *Message) {
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

	// 广播聊天消息
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

// handleDisconnect 处理断开连接
func (s *Server) handleDisconnect(player *Player) {
	player.Status = "offline"
	player.Connection = nil

	if player.Room != nil {
		// 通知房间内其他玩家
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

// broadcastToRoom 向房间内所有玩家广播消息
func (s *Server) broadcastToRoom(room *GameRoom, msg Message, excludePlayerID string) {
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	for playerID, player := range room.Players {
		if playerID != excludePlayerID && player.Connection != nil {
			player.Connection.WriteJSON(msg)
		}
	}
}

// sendError 发送错误消息
func (s *Server) sendError(conn *websocket.Conn, message string) {
	errorMsg := Message{
		Type: MsgTypeError,
		Data: map[string]interface{}{
			"message": message,
		},
		Timestamp: time.Now(),
	}
	conn.WriteJSON(errorMsg)
}

// sendErrorToPlayer 向玩家发送错误消息
func (s *Server) sendErrorToPlayer(player *Player, message string) {
	if player.Connection != nil {
		s.sendError(player.Connection, message)
	}
}