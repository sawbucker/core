// Package sessions
package sessions

import (
	"sync"

	clog "github.com/ShoshinNikita/log/v2"
	"github.com/gorilla/websocket"

	"github.com/tags-drive/core/cmd"
)

type WebSocketSessionStorage struct {
	// sessions stores all WebSocketSession. Keys are RemoteAddr of requests
	sessions map[string]*cmd.WebSocketSession
	mu       *sync.RWMutex

	logger *clog.Logger
}

func NewSessionsStorage(l *clog.Logger) *WebSocketSessionStorage {
	return &WebSocketSessionStorage{
		sessions: make(map[string]*cmd.WebSocketSession),
		mu:       new(sync.RWMutex),
		logger:   l,
	}
}

func (storage *WebSocketSessionStorage) AddSession(s *cmd.WebSocketSession) {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	storage.logger.Infof("add WS session: '%s'\n", s.RemoteAddr)
	storage.sessions[s.RemoteAddr] = s

	go func() {
		for {
			msgType, _, err := s.Conn.ReadMessage()
			if msgType == websocket.CloseMessage || err != nil {

				addr := s.RemoteAddr
				storage.logger.Infof("remove session '%s'\n", addr)

				storage.mu.Lock()
				delete(storage.sessions, s.RemoteAddr)
				storage.mu.Unlock()

				return
			}
		}
	}()
}

func (storage *WebSocketSessionStorage) Broadcast(msg []byte) {
	storage.mu.RLock()
	defer storage.mu.RUnlock()

	var err error
	for addr, session := range storage.sessions {
		err = session.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			storage.logger.Errorf("can't send message through WS to '%s'\n", addr)
		}
	}
}

func (storage *WebSocketSessionStorage) Shutdown() error {
	// Wait for all other locks
	storage.mu.Lock()
	storage.mu.Unlock()

	return nil
}
