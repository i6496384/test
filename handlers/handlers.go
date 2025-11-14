package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wireguard-web-manager/models"

	"github.com/gin-gonic/gin"
)

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

	server.ID = models.GenerateServerID()
	server.CreatedAt = time.Now()
	server.UpdatedAt = time.Now()
	server.IsActive = true

	// Генерация ключей (заглушка - в реальности нужна криптография)
	server.PrivateKey = generateKey()
	server.PublicKey = generateKey()

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

	server.ID = id
	server.UpdatedAt = time.Now()

	models.GlobalStorage.UpdateServer(&server)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    server,
	})
}

// DeleteServer удаление сервера
func DeleteServer(c *gin.Context) {
	id := c.Param("id")
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

	client.ID = models.GenerateClientID()
	client.CreatedAt = time.Now()
	client.UpdatedAt = time.Now()
	client.IsActive = true
	client.IsDisabled = false
	client.Downloaded = false

	// Генерация ключей (заглушка)
	client.PrivateKey = generateKey()
	client.PublicKey = generateKey()

	// Генерация IP адреса для клиента
	client.AllowedIPs = generateClientIP()

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
	config := generateWireGuardConfig(server, client)

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

	client.IsDisabled = true
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

	client.IsDisabled = false
	models.GlobalStorage.UpdateClient(client)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Клиент включен",
	})
}

// DeleteClient удаление клиента
func DeleteClient(c *gin.Context) {
	id := c.Param("id")
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

func generateKey() string {
	// Заглушка для генерации ключей
	// В реальности нужно использовать WireGuard криптографию
	return "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl0123456789+/"
}

func generateClientIP() string {
	// Простая генерация IP адреса (заглушка)
	// В реальности нужна проверка на уникальность
	return "10.0.0." + strconv.Itoa(time.Now().Second()%254+2)
}

func generateWireGuardConfig(server *models.Server, client *models.Client) string {
	var config strings.Builder

	config.WriteString("[Interface]\n")
	config.WriteString("PrivateKey = " + client.PrivateKey + "\n")
	config.WriteString("Address = " + client.AllowedIPs + "/32\n")
	config.WriteString("DNS = " + server.DNS + "\n\n")

	config.WriteString("[Peer]\n")
	config.WriteString("PublicKey = " + server.PublicKey + "\n")
	config.WriteString("Endpoint = " + server.Endpoint + "\n")
	config.WriteString("AllowedIPs = " + server.AllowedIPs + "\n")
	config.WriteString("PersistentKeepalive = 25\n")

	return config.String()
}
