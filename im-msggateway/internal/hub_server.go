package internal

import (
	"sync"
)

type HubServer struct {
	WsServer WsServer
	rooms    map[string]*Room
	roomMu   sync.RWMutex
}

type Room struct {
	ID          string
	connections map[string]*WsConnection
	mu          sync.RWMutex
}

func NewHubServer(wsServer WsServer) *HubServer {
	return &HubServer{
		WsServer: wsServer,
		rooms:    make(map[string]*Room),
	}
}

func (h *HubServer) JoinRoom(roomID string, conn *WsConnection) {
	h.roomMu.Lock()
	defer h.roomMu.Unlock()

	room, exists := h.rooms[roomID]
	if !exists {
		room = &Room{
			ID:          roomID,
			connections: make(map[string]*WsConnection),
		}
		h.rooms[roomID] = room
	}

	room.mu.Lock()
	room.connections[conn.ID()] = conn
	room.mu.Unlock()
}

func (h *HubServer) LeaveRoom(roomID string, conn *WsConnection) {
	h.roomMu.RLock()
	room, exists := h.rooms[roomID]
	h.roomMu.RUnlock()

	if !exists {
		return
	}

	room.mu.Lock()
	delete(room.connections, conn.ID())

	if len(room.connections) == 0 {
		h.roomMu.Lock()
		delete(h.rooms, roomID)
		h.roomMu.Unlock()
	}
	room.mu.Unlock()
}

func (h *HubServer) BroadcastToRoom(roomID string, message []byte) {
	h.roomMu.RLock()
	room, exists := h.rooms[roomID]
	h.roomMu.RUnlock()

	if !exists {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for _, conn := range room.connections {
		conn.Send(message)
	}
}

func (h *HubServer) GetRoomConnections(roomID string) []*WsConnection {
	h.roomMu.RLock()
	room, exists := h.rooms[roomID]
	h.roomMu.RUnlock()

	if !exists {
		return nil
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	conns := make([]*WsConnection, 0, len(room.connections))
	for _, conn := range room.connections {
		conns = append(conns, conn)
	}
	return conns
}

func (h *HubServer) RoomCount() int {
	h.roomMu.RLock()
	defer h.roomMu.RUnlock()
	return len(h.rooms)
}
