package webhook

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/impa-hub/internal/chatwoot"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/models"
	"github.com/impa-hub/internal/typebot"
)

func RegisterRoutes(r *gin.Engine) {
	// Webhook do Evolution GO -> IMPA HUB (sem auth)
	r.POST("/webhook/:instanceId", handleEvoWebhook)
}

// handleEvoWebhook recebe eventos do Evolution GO
func handleEvoWebhook(c *gin.Context) {
	instanceID, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var payload chatwoot.EvoWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
		return
	}

	log.Printf("[IMPA HUB] Webhook recebido: evento=%s instância=%s", payload.Event, instanceID)

	// Processa assíncronamente
	go func() {
		// Atualiza status da instância baseado no evento
		updateInstanceStatus(instanceID, payload.Event)

		// Encaminha para as integrações configuradas
		chatwoot.ProcessEvoWebhook(instanceID, payload)

		// Encaminha para o Typebot
		typebotPayload := typebot.EvoWebhookPayload{
			Event:         payload.Event,
			InstanceToken: payload.InstanceToken,
			InstanceID:    payload.InstanceID,
			InstanceName:  payload.InstanceName,
			Data:          payload.Data,
		}
		typebot.ProcessEvoWebhook(instanceID, typebotPayload)

		// Loga o evento
		wlog := models.WebhookLog{
			InstanceID: instanceID,
			Event:      payload.Event,
			Direction:  "incoming",
			Status:     "processed",
		}
		database.DB.Create(&wlog)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func updateInstanceStatus(instanceID uuid.UUID, event string) {
	switch event {
	case "Connected", "PairSuccess":
		database.DB.Model(&models.Instance{}).Where("id = ?", instanceID).Update("status", models.StatusConnected)
	case "LoggedOut", "Disconnected":
		database.DB.Model(&models.Instance{}).Where("id = ?", instanceID).Update("status", models.StatusDisconnected)
	case "QRCode":
		database.DB.Model(&models.Instance{}).Where("id = ?", instanceID).Update("status", models.StatusQRCode)
	}
}
