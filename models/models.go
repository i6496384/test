package models

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Server представляет конфигурацию WireGuard сервера
type Server struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ListenPort  int       `json:"listen_port"`
	PrivateKey  string    `json:"private_key"`
	PublicKey   string    `json:"public_key"`
	Network     string    `json:"network"` // например, 10.0.0.0/24
	DNS         string    `json:"dns"`     // например, 8.8.8.8
	AllowedIPs  string    `json:"allowed_ips"` // например, 0.0.0.0/0
	Endpoint    string    `json:"endpoint"` // внешний IP:порт сервера
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Client представляет клиента WireGuard
type Client struct {
	ID          string    `json:"id"`
	ServerID    string    `json:"server_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	PrivateKey  string    `json:"private_key"`
	PublicKey   string    `json:"public_key"`
	AllowedIPs  string    `json:"allowed_ips"` // IP адрес клиента в сети сервера
	IsActive    bool      `json:"is_active"`
	IsDisabled  bool      `json:"is_disabled"`
	Downloaded  bool      `json:"downloaded"` // скачал ли клиент конфиг
	DownloadAt  *time.Time `json:"download_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Stats представляет статистику по клиентам
type Stats struct {
	TotalClients    int `json:"total_clients"`
	ActiveClients   int `json:"active_clients"`
	DisabledClients int `json:"disabled_clients"`
	DownloadedCount int `json:"downloaded_count"`
}

// Storage представляет хранилище данных
type Storage struct {
	Servers map[string]*Server
	Clients map[string]*Client
	mu      sync.RWMutex
}

// Глобальное хранилище данных
var GlobalStorage *Storage

// InitStorage инициализирует глобальное хранилище
func InitStorage() {
	GlobalStorage = &Storage{
		Servers: make(map[string]*Server),
		Clients: make(map[string]*Client),
	}

	GlobalStorage.Servers[GenerateServerID()] = &Server{
		ID:          GenerateServerID(),
		Name:        "WireGuard Server",
		ListenPort:  10000,
		PrivateKey:  "PrivateKey",
		PublicKey:   "PublicKey",
		Network:     "10.0.0.0/24",
		DNS:         "8.8.8.8",
		AllowedIPs:  "0.0.0.0/0",
		Endpoint:    "10.0.0.1:10000",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// GenerateServerID генерирует уникальный ID для сервера
func GenerateServerID() string {
	return uuid.New().String()
}

// GenerateClientID генерирует уникальный ID для клиента
func GenerateClientID() string {
	return uuid.New().String()
}

// AddServer добавляет сервер в хранилище
func (s *Storage) AddServer(server *Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Servers[server.ID] = server
}

// GetServer получает сервер по ID
func (s *Storage) GetServer(id string) (*Server, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	server, exists := s.Servers[id]
	return server, exists
}

// UpdateServer обновляет сервер
func (s *Storage) UpdateServer(server *Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	server.UpdatedAt = time.Now()
	s.Servers[server.ID] = server
}

// DeleteServer удаляет сервер
func (s *Storage) DeleteServer(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Servers, id)
}

// AddClient добавляет клиента в хранилище
func (s *Storage) AddClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Clients[client.ID] = client
}

// GetClient получает клиента по ID
func (s *Storage) GetClient(id string) (*Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, exists := s.Clients[id]
	return client, exists
}

// UpdateClient обновляет клиента
func (s *Storage) UpdateClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	client.UpdatedAt = time.Now()
	s.Clients[client.ID] = client
}

// DeleteClient удаляет клиента
func (s *Storage) DeleteClient(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Clients, id)
}

// GetAllClients получает всех клиентов
func (s *Storage) GetAllClients() map[string]*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Clients
}

// GetClientsByServerID получает клиентов по ID сервера
func (s *Storage) GetClientsByServerID(serverID string) map[string]*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*Client)
	for id, client := range s.Clients {
		if client.ServerID == serverID {
			result[id] = client
		}
	}
	return result
}

// GetStats возвращает статистику
func (s *Storage) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var stats Stats
	for _, client := range s.Clients {
		stats.TotalClients++
		if client.IsActive && !client.IsDisabled {
			stats.ActiveClients++
		}
		if client.IsDisabled {
			stats.DisabledClients++
		}
		if client.Downloaded {
			stats.DownloadedCount++
		}
	}
	
	return stats
}