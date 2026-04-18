package instance

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/impa-hub/internal/evoclient"
	"github.com/impa-hub/internal/middleware"
)

func RegisterRoutes(r *gin.RouterGroup) {
	instances := r.Group("/instances")
	instances.Use(middleware.AuthMiddleware())
	{
		instances.GET("", handleListInstances)
		instances.POST("", handleCreateInstance)
		instances.GET("/:id", handleGetInstance)
		instances.DELETE("/:id", handleDeleteInstance)

		// Conexão
		instances.POST("/:id/connect", handleConnectInstance)
		instances.GET("/:id/status", handleGetStatus)
		instances.GET("/:id/qr", handleGetQRCode)
		instances.POST("/:id/disconnect", handleDisconnectInstance)
		instances.POST("/:id/logout", handleLogoutInstance)
		instances.POST("/:id/reconnect", handleReconnectInstance)

		// Mensagens
		instances.POST("/:id/send/text", handleSendText)
		instances.POST("/:id/send/media", handleSendMedia)

		// Advanced Settings
		instances.GET("/:id/advanced-settings", handleGetAdvancedSettings)
		instances.PUT("/:id/advanced-settings", handleUpdateAdvancedSettings)

		// Pairing
		instances.POST("/:id/pair", handlePairInstance)
	}
}

func handleListInstances(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	instances, err := ListInstances(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": instances})
}

func handleCreateInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	resp, err := CreateInstance(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

func handleGetInstance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	resp, err := GetInstance(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func handleDeleteInstance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := DeleteInstance(userID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Instância removida"})
}

func handleConnectInstance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var req ConnectInstanceRequest
	_ = c.ShouldBindJSON(&req)

	userID := middleware.GetCurrentUserID(c)
	data, err := ConnectInstance(userID, id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Instância conectada", "data": data})
}

func handleGetStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	status, err := GetInstanceStatus(userID, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
}

func handleGetQRCode(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	data, err := GetQRCode(userID, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func handleDisconnectInstance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := DisconnectInstance(userID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Instância desconectada"})
}

func handleLogoutInstance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := LogoutInstance(userID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout realizado"})
}

func handleReconnectInstance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := ReconnectInstance(userID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reconectando instância"})
}

type SendTextBody struct {
	Number string `json:"number" binding:"required"`
	Text   string `json:"text" binding:"required"`
}

func handleSendText(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var body SendTextBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	resp, err := SendText(userID, id, body.Number, body.Text)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func handleSendMedia(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var body evoclient.SendMediaRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	resp, err := SendMedia(userID, id, body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// ==================== Advanced Settings ====================

func handleGetAdvancedSettings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	data, err := GetAdvancedSettings(userID, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func handleUpdateAdvancedSettings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Erro ao ler body"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	data, err := UpdateAdvancedSettings(userID, id, json.RawMessage(body))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configurações atualizadas", "data": data})
}

// ==================== Pairing ====================

type PairBody struct {
	Phone string `json:"phone" binding:"required"`
}

func handlePairInstance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var body PairBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Telefone é obrigatório"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	data, err := PairInstance(userID, id, body.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pairing code gerado", "data": data})
}
