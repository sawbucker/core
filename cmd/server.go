package cmd

import (
	"github.com/gorilla/websocket"
)

// ServerInterface provides methods for interactions web server
type ServerInterface interface {
	Start() error

	// Shutdown gracefully shutdowns server
	Shutdown() error
}

// AuthServiceInterface provides methods for auth users
type AuthServiceInterface interface {
	// GenerateToken generates a new token. GenerateToken doesn't add new token, just return it!
	GenerateToken() string

	// AddToken adds passed token into storage
	AddToken(token string)

	// CheckToken returns true if token is in storage
	CheckToken(token string) bool

	// DeleteToken deletes token from a storage
	DeleteToken(token string)

	// Shutdown gracefully shutdown FileStorage
	Shutdown() error
}

// SessionStorageInterface provides methods for auth users
type WebSocketSessionStorageInterface interface {
	AddSession(s *WebSocketSession)

	Broadcast(msg []byte)

	Shutdown() error
}

type WebSocketSession struct {
	Conn       *websocket.Conn
	RemoteAddr string
}
