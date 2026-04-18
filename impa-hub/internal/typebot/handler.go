package typebot

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/impa-hub/internal/middleware"
)

func RegisterRoutes(r *gin.RouterGroup) {
	tb := r.Group("/integrations/typebot")
	tb.Use(middleware.AuthMiddleware())
	{
		// CRUD de bots (múltiplos por instância)
		tb.POST("/create", handleCreateConfig)
		tb.GET("/find", handleFindConfigs)
		tb.GET("/find/:instanceId", handleFindConfigsByInstance)
		tb.GET("/fetch/:typebotId", handleFetchConfig)
		tb.PUT("/update/:typebotId", handleUpdateConfig)
		tb.DELETE("/delete/:typebotId", handleDeleteConfig)

		// Settings globais por instância
		tb.POST("/settings/:instanceId", handleSetSettings)
		tb.GET("/fetchSettings/:instanceId", handleFetchSettings)

		// IgnoreJid
		tb.POST("/ignoreJid/:instanceId", handleIgnoreJid)

		// Sessões
		tb.GET("/fetchSessions/:instanceId", handleGetSessions)
		tb.POST("/changeStatus/:instanceId", handleChangeStatus)
		tb.POST("/start/:instanceId", handleStartTypebot)
	}
}

func handleCreateConfig(c *gin.Context) {
	var req CreateTypebotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	resp, err := CreateTypebotConfig(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func handleFindConfigs(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	configs, err := FindTypebotConfigs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func handleFindConfigsByInstance(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	configs, err := FindTypebotConfigsByInstance(userID, instanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func handleFetchConfig(c *gin.Context) {
	typebotID, err := uuid.Parse(c.Param("typebotId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	resp, err := FetchTypebotConfig(userID, typebotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func handleUpdateConfig(c *gin.Context) {
	typebotID, err := uuid.Parse(c.Param("typebotId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	var req UpdateTypebotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	if err := UpdateTypebotConfig(userID, typebotID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Configuração atualizada"})
}

func handleDeleteConfig(c *gin.Context) {
	typebotID, err := uuid.Parse(c.Param("typebotId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	if err := DeleteTypebotConfig(userID, typebotID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Configuração Typebot removida"})
}

func handleSetSettings(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	var req SetSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	resp, err := SetSettings(userID, instanceID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func handleFetchSettings(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	resp, err := FetchSettings(userID, instanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func handleIgnoreJid(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	var req IgnoreJidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	if err := IgnoreJid(userID, instanceID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "IgnoreJid atualizado"})
}

func handleGetSessions(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	sessions, err := GetSessions(userID, instanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sessions})
}

func handleChangeStatus(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	var req ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}
	userID := middleware.GetCurrentUserID(c)
	if err := ChangeSessionStatus(userID, instanceID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status atualizado"})
}

func handleStartTypebot(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	var req StartTypebotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}
	session, err := StartTypebotManual(instanceID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}
