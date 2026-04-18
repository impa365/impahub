package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/impa-hub/internal/middleware"
)

func RegisterRoutes(r *gin.RouterGroup) {
	servers := r.Group("/servers")
	servers.Use(middleware.AuthMiddleware())
	{
		servers.GET("", handleListServers)
		servers.POST("", handleCreateServer)
		servers.GET("/:id", handleGetServer)
		servers.PUT("/:id", handleUpdateServer)
		servers.DELETE("/:id", handleDeleteServer)
		servers.POST("/:id/test", handleTestConnection)
	}
}

func handleListServers(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	servers, err := ListServers(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": servers})
}

func handleCreateServer(c *gin.Context) {
	var req CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	srv, err := CreateServer(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": srv})
}

func handleGetServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	srv, err := GetServer(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": srv})
}

func handleUpdateServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var req UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := UpdateServer(userID, id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Servidor atualizado"})
}

func handleDeleteServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := DeleteServer(userID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Servidor removido"})
}

func handleTestConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := TestServerConnection(userID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Falha na conexão: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Conexão bem-sucedida"})
}
