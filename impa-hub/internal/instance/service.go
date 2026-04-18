package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/impa-hub/internal/config"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/evoclient"
	"github.com/impa-hub/internal/models"
	"gorm.io/gorm"
)

// Referência ao pacote server para funções auxiliares
var _ = fmt.Sprintf

type CreateInstanceRequest struct {
	EvoServerID  uuid.UUID `json:"server_id" binding:"required"`
	InstanceName string    `json:"instance_name" binding:"required"`
	Phone        string    `json:"phone,omitempty"`
}

type ConnectInstanceRequest struct {
	Phone string `json:"phone,omitempty"`
}

type InstanceResponse struct {
	ID                uuid.UUID             `json:"id"`
	EvoServerID       uuid.UUID             `json:"server_id"`
	EvoServerName     string                `json:"server_name"`
	EvoInstanceID     string                `json:"evo_instance_id"`
	EvoInstanceName   string                `json:"instance_name"`
	Status            models.InstanceStatus `json:"connection_status"`
	Phone             string                `json:"phone,omitempty"`
	PushName          string                `json:"push_name,omitempty"`
	WebhookConfigured bool                  `json:"webhook_configured"`
	HasChatwoot       bool                  `json:"has_chatwoot"`
	HasTypebot        bool                  `json:"has_typebot"`
}

// CreateInstance cria uma instância no EVO GO e registra no IMPA HUB
func CreateInstance(userID uuid.UUID, req CreateInstanceRequest) (*InstanceResponse, error) {
	// Verifica quota
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("usuário não encontrado")
	}

	var count int64
	database.DB.Model(&models.Instance{}).Where("user_id = ?", userID).Count(&count)
	if int(count) >= user.MaxInstances {
		return nil, fmt.Errorf("limite de instâncias atingido (%d/%d)", count, user.MaxInstances)
	}

	// Verifica se o servidor pertence ao usuário
	var srv models.EvoServer
	if err := database.DB.Where("id = ? AND user_id = ?", req.EvoServerID, userID).First(&srv).Error; err != nil {
		return nil, errors.New("servidor não encontrado")
	}

	// Cria instância no Evolution GO
	evoClient := evoclient.New(srv.URL, srv.APIKey)
	instanceToken := uuid.New().String()
	evoResp, err := evoClient.CreateInstance(evoclient.CreateInstanceRequest{
		Name:  req.InstanceName,
		Token: instanceToken,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao criar instância no EVO GO: %v", err)
	}

	// Salva no IMPA HUB
	instance := models.Instance{
		UserID:          userID,
		EvoServerID:     req.EvoServerID,
		EvoInstanceID:   evoResp.InstanceID,
		EvoInstanceName: evoResp.InstanceName,
		EvoToken:        evoResp.Token,
		Status:          models.StatusDisconnected,
		Phone:           req.Phone,
	}

	if err := database.DB.Create(&instance).Error; err != nil {
		// Tenta deletar a instância criada no EVO se falhar
		_ = evoClient.DeleteInstance(evoResp.InstanceID)
		return nil, err
	}

	return &InstanceResponse{
		ID:                instance.ID,
		EvoServerID:       srv.ID,
		EvoServerName:     srv.Name,
		EvoInstanceID:     evoResp.InstanceID,
		EvoInstanceName:   evoResp.InstanceName,
		Status:            models.StatusDisconnected,
		Phone:             req.Phone,
		WebhookConfigured: false,
		HasChatwoot:       false,
	}, nil
}

// ConnectInstance conecta uma instância e configura o webhook apontando para IMPA HUB
func ConnectInstance(userID, instanceID uuid.UUID, req ConnectInstanceRequest) (interface{}, error) {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return nil, err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)

	// Webhook URL apontando para o IMPA HUB
	webhookURL := fmt.Sprintf("%s/webhook/%s", config.AppConfig.BaseURL, inst.ID.String())

	connectReq := evoclient.ConnectInstanceRequest{
		WebhookURL: webhookURL,
		Subscribe:  []string{"ALL"},
		Immediate:  true,
		Phone:      req.Phone,
	}

	data, err := evoClient.ConnectInstance(inst.EvoToken, connectReq)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar: %v", err)
	}

	// Atualiza status e webhook
	database.DB.Model(inst).Updates(map[string]interface{}{
		"status":             models.StatusConnecting,
		"webhook_configured": true,
	})

	log.Printf("[IMPA HUB] Instância %s conectada com webhook: %s", inst.EvoInstanceName, webhookURL)

	return data, nil
}

func GetInstanceStatus(userID, instanceID uuid.UUID) (*evoclient.InstanceStatusData, error) {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return nil, err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	status, err := evoClient.GetInstanceStatus(inst.EvoToken)
	if err != nil {
		return nil, err
	}

	// Atualiza status local
	newStatus := models.StatusDisconnected
	if status.Connected && status.LoggedIn {
		newStatus = models.StatusConnected
	} else if status.Connected {
		newStatus = models.StatusConnecting
	}

	updates := map[string]interface{}{"status": newStatus}
	if status.Name != "" {
		updates["push_name"] = status.Name
	}
	if status.JID != "" {
		updates["phone"] = status.JID
	}
	database.DB.Model(inst).Updates(updates)

	return status, nil
}

func GetQRCode(userID, instanceID uuid.UUID) (*evoclient.QRCodeData, error) {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return nil, err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	return evoClient.GetQRCode(inst.EvoToken)
}

func DisconnectInstance(userID, instanceID uuid.UUID) error {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	if err := evoClient.DisconnectInstance(inst.EvoToken); err != nil {
		return err
	}

	database.DB.Model(inst).Update("status", models.StatusDisconnected)
	return nil
}

func LogoutInstance(userID, instanceID uuid.UUID) error {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	if err := evoClient.LogoutInstance(inst.EvoToken); err != nil {
		return err
	}

	database.DB.Model(inst).Updates(map[string]interface{}{
		"status": models.StatusDisconnected,
		"phone":  "",
	})
	return nil
}

func DeleteInstance(userID, instanceID uuid.UUID) error {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return err
	}

	// Remove do EVO GO
	evoClient := evoclient.New(srv.URL, srv.APIKey)
	_ = evoClient.DeleteInstance(inst.EvoInstanceID) // Ignora erro se já não existe

	// Remove configurações de Chatwoot
	database.DB.Where("instance_id = ?", instanceID).Delete(&models.ChatwootConfig{})
	database.DB.Where("chatwoot_config_id IN (SELECT id FROM chatwoot_configs WHERE instance_id = ?)", instanceID).Delete(&models.ChatwootConversation{})
	database.DB.Where("chatwoot_config_id IN (SELECT id FROM chatwoot_configs WHERE instance_id = ?)", instanceID).Delete(&models.ChatwootMessageMap{})

	// Remove configurações de Typebot
	database.DB.Where("typebot_config_id IN (SELECT id FROM typebot_configs WHERE instance_id = ?)", instanceID).Delete(&models.TypebotSession{})
	database.DB.Where("instance_id = ?", instanceID).Delete(&models.TypebotConfig{})

	// Remove instância
	return database.DB.Delete(inst).Error
}

func ListInstances(userID uuid.UUID) ([]InstanceResponse, error) {
	var instances []models.Instance
	if err := database.DB.Where("user_id = ?", userID).Find(&instances).Error; err != nil {
		return nil, err
	}

	// Agrupa instâncias por servidor para reutilizar clients
	serverClients := make(map[uuid.UUID]*evoclient.Client)
	serverNames := make(map[uuid.UUID]string)

	var result []InstanceResponse
	for _, inst := range instances {
		// Cache do servidor e client
		if _, ok := serverClients[inst.EvoServerID]; !ok {
			var srv models.EvoServer
			database.DB.First(&srv, inst.EvoServerID)
			serverClients[inst.EvoServerID] = evoclient.New(srv.URL, srv.APIKey)
			serverNames[inst.EvoServerID] = srv.Name
		}

		// Consulta status real no Evolution GO
		evoClient := serverClients[inst.EvoServerID]
		status, err := evoClient.GetInstanceStatus(inst.EvoToken)
		if err == nil {
			newStatus := models.StatusDisconnected
			if status.Connected && status.LoggedIn {
				newStatus = models.StatusConnected
			} else if status.Connected {
				newStatus = models.StatusConnecting
			}

			updates := map[string]interface{}{"status": newStatus}
			if status.Name != "" {
				updates["push_name"] = status.Name
			}
			if status.JID != "" {
				updates["phone"] = status.JID
			}
			database.DB.Model(&inst).Updates(updates)
			inst.Status = newStatus
			if status.Name != "" {
				inst.PushName = status.Name
			}
			if status.JID != "" {
				inst.Phone = status.JID
			}
		}

		var hasChatwoot bool
		var cwCount int64
		database.DB.Model(&models.ChatwootConfig{}).Where("instance_id = ? AND enabled = true", inst.ID).Count(&cwCount)
		hasChatwoot = cwCount > 0

		var hasTypebot bool
		var tbCount int64
		database.DB.Model(&models.TypebotConfig{}).Where("instance_id = ? AND enabled = true", inst.ID).Count(&tbCount)
		hasTypebot = tbCount > 0

		result = append(result, InstanceResponse{
			ID:                inst.ID,
			EvoServerID:       inst.EvoServerID,
			EvoServerName:     serverNames[inst.EvoServerID],
			EvoInstanceID:     inst.EvoInstanceID,
			EvoInstanceName:   inst.EvoInstanceName,
			Status:            inst.Status,
			Phone:             inst.Phone,
			PushName:          inst.PushName,
			WebhookConfigured: inst.WebhookConfigured,
			HasChatwoot:       hasChatwoot,
			HasTypebot:        hasTypebot,
		})
	}
	return result, nil
}

func GetInstance(userID, instanceID uuid.UUID) (*InstanceResponse, error) {
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("instância não encontrada")
		}
		return nil, err
	}

	var srv models.EvoServer
	database.DB.First(&srv, inst.EvoServerID)

	var cwCount int64
	database.DB.Model(&models.ChatwootConfig{}).Where("instance_id = ? AND enabled = true", inst.ID).Count(&cwCount)

	var tbCount int64
	database.DB.Model(&models.TypebotConfig{}).Where("instance_id = ? AND enabled = true", inst.ID).Count(&tbCount)

	return &InstanceResponse{
		ID:                inst.ID,
		EvoServerID:       inst.EvoServerID,
		EvoServerName:     srv.Name,
		EvoInstanceID:     inst.EvoInstanceID,
		EvoInstanceName:   inst.EvoInstanceName,
		Status:            inst.Status,
		Phone:             inst.Phone,
		PushName:          inst.PushName,
		WebhookConfigured: inst.WebhookConfigured,
		HasChatwoot:       cwCount > 0,
		HasTypebot:        tbCount > 0,
	}, nil
}

func ReconnectInstance(userID, instanceID uuid.UUID) error {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	return evoClient.ReconnectInstance(inst.EvoToken)
}

// SendText envia mensagem de texto pela instância
func SendText(userID, instanceID uuid.UUID, number, text string) (interface{}, error) {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return nil, err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	return evoClient.SendText(inst.EvoToken, evoclient.SendTextRequest{
		Number: number,
		Text:   text,
	})
}

// SendMedia envia mídia pela instância
func SendMedia(userID, instanceID uuid.UUID, req evoclient.SendMediaRequest) (interface{}, error) {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return nil, err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	return evoClient.SendMedia(inst.EvoToken, req)
}

// ==================== Helpers ====================

func getInstanceWithServer(userID, instanceID uuid.UUID) (*models.Instance, *models.EvoServer, error) {
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("instância não encontrada")
		}
		return nil, nil, err
	}

	var srv models.EvoServer
	if err := database.DB.First(&srv, inst.EvoServerID).Error; err != nil {
		return nil, nil, errors.New("servidor não encontrado")
	}

	return &inst, &srv, nil
}

// GetInstanceByID retorna uma instância pelo ID (sem verificar userID - uso interno)
func GetInstanceByID(instanceID uuid.UUID) (*models.Instance, error) {
	var inst models.Instance
	if err := database.DB.First(&inst, instanceID).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

// GetServerByID retorna um servidor pelo ID (uso interno)
func GetServerByID(serverID uuid.UUID) (*models.EvoServer, error) {
	var srv models.EvoServer
	if err := database.DB.First(&srv, serverID).Error; err != nil {
		return nil, err
	}
	return &srv, nil
}

// ==================== Advanced Settings ====================

// GetAdvancedSettings obtém configurações avançadas da instância no EVO GO
func GetAdvancedSettings(userID, instanceID uuid.UUID) (json.RawMessage, error) {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return nil, err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	return evoClient.GetAdvancedSettings(inst.EvoInstanceID, inst.EvoToken)
}

// UpdateAdvancedSettings atualiza configurações avançadas da instância no EVO GO
func UpdateAdvancedSettings(userID, instanceID uuid.UUID, settings json.RawMessage) (json.RawMessage, error) {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return nil, err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	return evoClient.UpdateAdvancedSettings(inst.EvoInstanceID, inst.EvoToken, settings)
}

// ==================== Pairing ====================

// PairInstance solicita código de pareamento via telefone
func PairInstance(userID, instanceID uuid.UUID, phone string) (interface{}, error) {
	inst, srv, err := getInstanceWithServer(userID, instanceID)
	if err != nil {
		return nil, err
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)
	data, err := evoClient.PairInstance(inst.EvoToken, phone)
	if err != nil {
		return nil, fmt.Errorf("erro ao solicitar pairing code: %v", err)
	}

	// Atualiza status
	database.DB.Model(inst).Updates(map[string]interface{}{
		"status":             models.StatusConnecting,
		"webhook_configured": true,
	})

	return data, nil
}
