package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"wireguard-web-manager/models"
	"wireguard-web-manager/wireguard"

	"github.com/gin-gonic/gin"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var wgService *wireguard.Service

func RegisterWireGuardService(service *wireguard.Service) {
	wgService = service
}

// Index главная страница
func Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "WireGuard Web Manager",
	})
}

// Dashboard страница панели управления
func Dashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Панель управления WireGuard",
	})
}

// GetServer получение сервера
func GetServer(c *gin.Context) {
	// Для простоты возвращаем первый сервер или создаем пустой
	server := &models.Server{}
	if len(models.GlobalStorage.Servers) > 0 {
		for _, s := range models.GlobalStorage.Servers {
			server = s
			break
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    server,
	})
}

// CreateServer создание сервера
func CreateServer(c *gin.Context) {
	var server models.Server
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Неверные данные: " + err.Error(),
		})
		return
	}

	if server.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Имя интерфейса обязательно",
		})
		return
	}

	server.ID = server.Name
	server.CreatedAt = time.Now()
	server.UpdatedAt = server.CreatedAt
	server.IsActive = true

	if server.PrivateKey == "" {
		key, err := wireguard.GeneratePrivateKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Не удалось сгенерировать ключ: " + err.Error(),
			})
			return
		}
		server.PrivateKey = key.String()
		server.PublicKey = key.PublicKey().String()
	} else {
		key, err := wgtypes.ParseKey(server.PrivateKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Неверный приватный ключ: " + err.Error(),
			})
			return
		}
		server.PrivateKey = key.String()
		server.PublicKey = key.PublicKey().String()
	}

	if wgService != nil {
		if err := wgService.ConfigureServer(server.ID, server.PrivateKey, server.ListenPort, true, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Не удалось настроить интерфейс WireGuard: " + err.Error(),
			})
			return
		}
	}

	models.GlobalStorage.AddServer(&server)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    server,
	})
}

// UpdateServer обновление сервера
func UpdateServer(c *gin.Context) {
	id := c.Param("id")
	var server models.Server
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Неверные данные: " + err.Error(),
		})
		return
	}

	existing, ok := models.GlobalStorage.GetServer(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Сервер не найден",
		})
		return
	}

	if server.Name == "" {
		server.Name = existing.Name
	}

	if server.Name != existing.Name {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Переименование интерфейса не поддерживается",
		})
		return
	}

	server.ID = existing.ID
	server.CreatedAt = existing.CreatedAt

	if server.PrivateKey == "" {
		server.PrivateKey = existing.PrivateKey
	}

	key, err := wgtypes.ParseKey(server.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Неверный приватный ключ: " + err.Error(),
		})
		return
	}
	server.PrivateKey = key.String()
	server.PublicKey = key.PublicKey().String()

	server.UpdatedAt = time.Now()

	if wgService != nil {
		if err := wgService.ConfigureServer(server.ID, server.PrivateKey, server.ListenPort, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Не удалось обновить конфигурацию WireGuard: " + err.Error(),
			})
			return
		}
	}

	models.GlobalStorage.UpdateServer(&server)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    server,
	})
}

// DeleteServer удаление сервера
func DeleteServer(c *gin.Context) {
	id := c.Param("id")

	if server, ok := models.GlobalStorage.GetServer(id); ok {
		if wgService != nil {
			if err := wgService.ConfigureServer(server.ID, "", 0, true, nil); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Не удалось очистить конфигурацию WireGuard: " + err.Error(),
				})
				return
			}
		}

		clients := models.GlobalStorage.GetClientsByServerID(id)
		for clientID := range clients {
			models.GlobalStorage.DeleteClient(clientID)
		}
	}

	models.GlobalStorage.DeleteServer(id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Сервер удален",
	})
}

// GetClients получение списка клиентов
func GetClients(c *gin.Context) {
	serverID := c.Query("server_id")

	var clients map[string]*models.Client
	if serverID != "" {
		clients = models.GlobalStorage.GetClientsByServerID(serverID)
	} else {
		clients = models.GlobalStorage.GetAllClients()
	}

	// Преобразование в слайс для JSON
	clientsList := make([]*models.Client, 0, len(clients))
	for _, client := range clients {
		clientsList = append(clientsList, client)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    clientsList,
	})
}

// CreateClient создание клиента
func CreateClient(c *gin.Context) {
	var client models.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Неверные данные: " + err.Error(),
		})
		return
	}

	if wgService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Сервис WireGuard недоступен",
		})
		return
	}

	server, ok := models.GlobalStorage.GetServer(client.ServerID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Сервер не найден",
		})
		return
	}

	var privateKey wgtypes.Key
	if client.PrivateKey == "" {
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Не удалось сгенерировать ключ: " + err.Error(),
			})
			return
		}
		privateKey = key
	} else {
		key, err := wgtypes.ParseKey(client.PrivateKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Неверный приватный ключ клиента: " + err.Error(),
			})
			return
		}
		privateKey = key
	}

	allowedInput := splitAllowedIPs(client.AllowedIPs)
	if len(allowedInput) == 0 {
		used := make(map[string]struct{})
		existing := models.GlobalStorage.GetClientsByServerID(server.ID)
		for _, item := range existing {
			addr := strings.TrimSpace(item.AllowedIPs)
			if addr == "" {
				continue
			}
			if idx := strings.Index(addr, "/"); idx > 0 {
				addr = addr[:idx]
			}
			used[addr] = struct{}{}
		}

		addr, err := wireguard.AllocateAddress(server.Network, used)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Не удалось выделить IP для клиента: " + err.Error(),
			})
			return
		}
		allowedInput = []string{addr}
	}

	allowedNetworks, err := wireguard.ParseAllowedIPs(allowedInput)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	keepalive := 25 * time.Second
	peerCfg := wgtypes.PeerConfig{
		PublicKey:                   privateKey.PublicKey(),
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  allowedNetworks,
		PersistentKeepaliveInterval: &keepalive,
	}

	if err := wgService.ConfigureServer(server.ID, "", 0, false, []wgtypes.PeerConfig{peerCfg}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Не удалось добавить клиента в WireGuard: " + err.Error(),
		})
		return
	}

	client.ID = models.GenerateClientID()
	client.ServerID = server.ID
	client.CreatedAt = time.Now()
	client.UpdatedAt = client.CreatedAt
	client.IsActive = true
	client.IsDisabled = false
	client.Downloaded = false
	client.PrivateKey = privateKey.String()
	client.PublicKey = privateKey.PublicKey().String()
	client.AllowedIPs = strings.Join(allowedInput, ", ")

	models.GlobalStorage.AddClient(&client)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    client,
	})
}

// DownloadClientConfig скачивание конфигурации клиента
func DownloadClientConfig(c *gin.Context) {
	id := c.Param("id")
	client, exists := models.GlobalStorage.GetClient(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Клиент не найден",
		})
		return
	}

	server, exists := models.GlobalStorage.GetServer(client.ServerID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Сервер не найден",
		})
		return
	}

	// Генерация конфигурации WireGuard
	config, err := generateWireGuardConfig(server, client)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Не удалось сформировать конфигурацию: " + err.Error(),
		})
		return
	}

	// Обновление статистики скачиваний
	client.Downloaded = true
	now := time.Now()
	client.DownloadAt = &now
	models.GlobalStorage.UpdateClient(client)

	c.Header("Content-Type", "text/plain")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.conf", client.Name))
	c.String(http.StatusOK, config)
}

// DisableClient отключение клиента
func DisableClient(c *gin.Context) {
	id := c.Param("id")
	client, exists := models.GlobalStorage.GetClient(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Клиент не найден",
		})
		return
	}

	if wgService != nil {
		if err := wgService.RemovePeer(client.ServerID, client.PublicKey); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Не удалось отключить клиента в WireGuard: " + err.Error(),
			})
			return
		}
	}

	client.IsDisabled = true
	client.IsActive = false
	models.GlobalStorage.UpdateClient(client)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Клиент отключен",
	})
}

// EnableClient включение клиента
func EnableClient(c *gin.Context) {
	id := c.Param("id")
	client, exists := models.GlobalStorage.GetClient(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Клиент не найден",
		})
		return
	}

	server, exists := models.GlobalStorage.GetServer(client.ServerID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Сервер не найден",
		})
		return
	}

	allowedInput := splitAllowedIPs(client.AllowedIPs)
	allowedNetworks, err := wireguard.ParseAllowedIPs(allowedInput)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	pubKey, err := wgtypes.ParseKey(client.PublicKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Некорректный публичный ключ клиента: " + err.Error(),
		})
		return
	}

	keepalive := 25 * time.Second
	peerCfg := wgtypes.PeerConfig{
		PublicKey:                   pubKey,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  allowedNetworks,
		PersistentKeepaliveInterval: &keepalive,
	}

	if wgService != nil {
		if err := wgService.ConfigureServer(server.ID, "", 0, false, []wgtypes.PeerConfig{peerCfg}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Не удалось включить клиента в WireGuard: " + err.Error(),
			})
			return
		}
	}

	client.IsDisabled = false
	client.IsActive = true
	models.GlobalStorage.UpdateClient(client)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Клиент включен",
	})
}

// DeleteClient удаление клиента
func DeleteClient(c *gin.Context) {
	id := c.Param("id")
	if client, exists := models.GlobalStorage.GetClient(id); exists {
		if wgService != nil {
			if err := wgService.RemovePeer(client.ServerID, client.PublicKey); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Не удалось удалить клиента из WireGuard: " + err.Error(),
				})
				return
			}
		}
	}
	models.GlobalStorage.DeleteClient(id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Клиент удален",
	})
}

// GetStats получение статистики
func GetStats(c *gin.Context) {
	stats := models.GlobalStorage.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// Вспомогательные функции

func generateWireGuardConfig(server *models.Server, client *models.Client) (string, error) {
	if client.PrivateKey == "" {
		return "", errors.New("у клиента отсутствует приватный ключ")
	}

	allowed := splitAllowedIPs(client.AllowedIPs)
	if len(allowed) == 0 {
		return "", errors.New("у клиента не настроены адреса")
	}

	var config strings.Builder

	config.WriteString("[Interface]\n")
	config.WriteString("PrivateKey = " + client.PrivateKey + "\n")
	config.WriteString("Address = " + strings.Join(ensureCIDR(allowed), ", ") + "\n")
	if server.DNS != "" {
		config.WriteString("DNS = " + server.DNS + "\n")
	}
	config.WriteString("\n")

	config.WriteString("[Peer]\n")
	config.WriteString("PublicKey = " + server.PublicKey + "\n")
	if server.Endpoint != "" {
		config.WriteString("Endpoint = " + server.Endpoint + "\n")
	}
	if server.AllowedIPs != "" {
		config.WriteString("AllowedIPs = " + server.AllowedIPs + "\n")
	}
	config.WriteString("PersistentKeepalive = 25\n")

	return config.String(), nil
}

func splitAllowedIPs(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func ensureCIDR(addresses []string) []string {
	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		if strings.Contains(addr, "/") {
			result = append(result, addr)
			continue
		}
		result = append(result, addr+"/32")
	}
	return result
}
