package chatwoot

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/impa-hub/internal/middleware"
)

func RegisterRoutes(r *gin.RouterGroup) {
	// Rotas autenticadas para gerenciar configurações Chatwoot
	cw := r.Group("/integrations/chatwoot")
	cw.Use(middleware.AuthMiddleware())
	{
		cw.GET("", handleListConfigs)
		cw.POST("/set", handleSetConfig)
		cw.GET("/:instanceId", handleGetConfig)
		cw.PUT("/:instanceId", handleUpdateConfig)
		cw.DELETE("/:instanceId", handleDeleteConfig)
	}
}

func RegisterWebhookRoutes(r *gin.Engine) {
	// Webhook do Chatwoot -> IMPA HUB (sem autenticação, como na Evolution API)
	r.POST("/chatwoot/webhook/:instanceId", handleChatwootWebhook)
}

func handleListConfigs(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	configs, err := ListChatwootConfigs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func handleSetConfig(c *gin.Context) {
	var req SetChatwootRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	resp, err := SetChatwootConfig(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func handleGetConfig(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	resp, err := GetChatwootConfig(userID, instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func handleUpdateConfig(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var req UpdateChatwootRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := UpdateChatwootConfig(userID, instanceID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuração atualizada"})
}

func handleDeleteConfig(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := DeleteChatwootConfig(userID, instanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuração Chatwoot removida"})
}

// handleChatwootWebhook recebe eventos do Chatwoot (sem auth - público)
func handleChatwootWebhook(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var payload ChatwootWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
		return
	}

	// Processa assíncronamente para não bloquear o Chatwoot
	go func() {
		if err := ProcessChatwootWebhook(instanceID, payload); err != nil {
			logWebhook(instanceID, payload.Event, "incoming", "error", err.Error())
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
