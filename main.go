package main

import (
	"log"
	"net/http"

	"wireguard-web-manager/handlers"
	"wireguard-web-manager/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// Инициализация хранилища данных
	models.InitStorage()

	// Настройка Gin
	r := gin.Default()

	// Загрузка статических файлов
	r.Static("/static", "./static")
	r.Static("/css", "./css")
	r.StaticFS("/uploads", http.Dir("./uploads"))

	// Загрузка HTML шаблонов
	r.LoadHTMLGlob("templates/*")

	// API маршруты
	api := r.Group("/api")
	{
		// Сервер
		api.GET("/server", handlers.GetServer)
		api.POST("/server", handlers.CreateServer)
		api.PUT("/server/:id", handlers.UpdateServer)
		api.DELETE("/server/:id", handlers.DeleteServer)

		// Клиенты
		api.GET("/clients", handlers.GetClients)
		api.POST("/clients", handlers.CreateClient)
		api.GET("/clients/:id/config", handlers.DownloadClientConfig)
		api.PUT("/clients/:id/disable", handlers.DisableClient)
		api.PUT("/clients/:id/enable", handlers.EnableClient)
		api.DELETE("/clients/:id", handlers.DeleteClient)

		// Статистика
		api.GET("/stats", handlers.GetStats)
	}

	// Веб-интерфейс маршруты
	r.GET("/", handlers.Index)
	r.GET("/dashboard", handlers.Dashboard)

	log.Println("Сервер запущен на порту :8080")
	r.Run(":8080")
}
