package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/impa-hub/internal/admin"
	"github.com/impa-hub/internal/auth"
	"github.com/impa-hub/internal/chatwoot"
	"github.com/impa-hub/internal/config"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/instance"
	"github.com/impa-hub/internal/server"
	"github.com/impa-hub/internal/typebot"
	"github.com/impa-hub/internal/webhook"
)

func main() {
	printBanner()

	// Carrega configuração
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("[IMPA HUB] JWT_SECRET é obrigatório! Configure no .env")
	}

	// Conecta ao banco de dados
	database.Connect(cfg)
	database.Migrate()

	// Cria super admin padrão
	if err := auth.CreateSuperAdminIfNotExists(cfg); err != nil {
		log.Printf("[IMPA HUB] Aviso ao criar super admin: %v", err)
	}

	// Configura Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware())
	r.Use(corsMiddleware(cfg))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "IMPA HUB",
			"version": "1.0.0",
		})
	})

	// API Routes (autenticadas)
	api := r.Group("/api/v1")
	{
		auth.RegisterRoutes(api)
		admin.RegisterRoutes(api)
		server.RegisterRoutes(api)
		instance.RegisterRoutes(api)
		chatwoot.RegisterRoutes(api)
		typebot.RegisterRoutes(api)
	}

	// Webhook Routes (sem autenticação)
	webhook.RegisterRoutes(r)
	chatwoot.RegisterWebhookRoutes(r)

	// Inicia servidor
	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	log.Printf("[IMPA HUB] Servidor iniciado em %s", addr)
	log.Printf("[IMPA HUB] Base URL: %s", cfg.BaseURL)
	log.Printf("[IMPA HUB] Admin: %s", cfg.AdminEmail)

	if err := r.Run(addr); err != nil {
		log.Fatalf("[IMPA HUB] Erro ao iniciar servidor: %v", err)
	}
}

func printBanner() {
	banner := `
 ██╗███╗   ███╗██████╗  █████╗     ██╗  ██╗██╗   ██╗██████╗ 
 ██║████╗ ████║██╔══██╗██╔══██╗    ██║  ██║██║   ██║██╔══██╗
 ██║██╔████╔██║██████╔╝███████║    ███████║██║   ██║██████╔╝
 ██║██║╚██╔╝██║██╔═══╝ ██╔══██║    ██╔══██║██║   ██║██╔══██╗
 ██║██║ ╚═╝ ██║██║     ██║  ██║    ██║  ██║╚██████╔╝██████╔╝
 ╚═╝╚═╝     ╚═╝╚═╝     ╚═╝  ╚═╝    ╚═╝  ╚═╝ ╚═════╝ ╚═════╝ 
                Integration Hub v1.0.0
      Evolution GO <-> Chatwoot | Typebot | ...
`
	fmt.Println(banner)
}

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[IMPA HUB] %s %s %d %v", c.Request.Method, path, status, latency)
	}
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origins := cfg.CORSOrigins
		if origins == "" {
			origins = "*"
		}

		c.Header("Access-Control-Allow-Origin", origins)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if strings.ToUpper(c.Request.Method) == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
