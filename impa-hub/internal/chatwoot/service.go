package chatwoot

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/impa-hub/internal/config"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/models"
	"gorm.io/gorm"
)

type SetChatwootRequest struct {
	InstanceID          uuid.UUID `json:"instance_id" binding:"required"`
	Enabled             bool      `json:"enabled"`
	URL                 string    `json:"url" binding:"required"`
	Token               string    `json:"token" binding:"required"`
	AccountID           string    `json:"account_id" binding:"required"`
	NameInbox           string    `json:"inbox_name" binding:"required"`
	SignMsg             bool      `json:"sign_msg"`
	SignDelimiter       string    `json:"sign_delimiter"`
	Number              string    `json:"number,omitempty"`
	ReopenConversation  bool      `json:"reopen_conversation"`
	ConversationPending bool      `json:"conversation_pending"`
	MergeBrazilContacts bool      `json:"merge_brazil_contacts"`
	ImportContacts      bool      `json:"import_contacts"`
	ImportMessages      bool      `json:"import_messages"`
	DaysLimitImport     int       `json:"days_limit_import"`
	AutoCreate          bool      `json:"auto_create"`
	Organization        string    `json:"organization,omitempty"`
	Logo                string    `json:"logo,omitempty"`
	GroupsIgnore        bool      `json:"groups_ignore"`
	IgnoreJids          string    `json:"ignore_jids,omitempty"`
}

type ChatwootConfigResponse struct {
	ID                  uuid.UUID `json:"id"`
	InstanceID          uuid.UUID `json:"instance_id"`
	InstanceName        string    `json:"instance_name"`
	Enabled             bool      `json:"enabled"`
	URL                 string    `json:"url"`
	Token               string    `json:"token"`
	AccountID           string    `json:"account_id"`
	NameInbox           string    `json:"inbox_name"`
	InboxID             int       `json:"inbox_id"`
	SignMsg             bool      `json:"sign_msg"`
	SignDelimiter       string    `json:"sign_delimiter"`
	ReopenConversation  bool      `json:"reopen_conversation"`
	ConversationPending bool      `json:"conversation_pending"`
	MergeBrazilContacts bool      `json:"merge_brazil_contacts"`
	AutoCreate          bool      `json:"auto_create"`
	IsActive            bool      `json:"is_active"`
	WebhookURL          string    `json:"webhook_url"`
	GroupsIgnore        bool      `json:"groups_ignore"`
	IgnoreJids          string    `json:"ignore_jids"`
}

type UpdateChatwootRequest struct {
	Enabled             *bool   `json:"enabled,omitempty"`
	URL                 *string `json:"url,omitempty"`
	Token               *string `json:"token,omitempty"`
	AccountID           *string `json:"account_id,omitempty"`
	NameInbox           *string `json:"inbox_name,omitempty"`
	SignMsg             *bool   `json:"sign_msg,omitempty"`
	SignDelimiter       *string `json:"sign_delimiter,omitempty"`
	Number              *string `json:"number,omitempty"`
	ReopenConversation  *bool   `json:"reopen_conversation,omitempty"`
	ConversationPending *bool   `json:"conversation_pending,omitempty"`
	MergeBrazilContacts *bool   `json:"merge_brazil_contacts,omitempty"`
	AutoCreate          *bool   `json:"auto_create,omitempty"`
	Organization        *string `json:"organization,omitempty"`
	Logo                *string `json:"logo,omitempty"`
}

// SetChatwootConfig cria ou atualiza a configuração Chatwoot para uma instância
func SetChatwootConfig(userID uuid.UUID, req SetChatwootRequest) (*ChatwootConfigResponse, error) {
	// Verifica permissão
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("usuário não encontrado")
	}

	if !user.CanUseChatwoot {
		return nil, errors.New("você não tem permissão para usar Chatwoot")
	}

	// Verifica instância
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", req.InstanceID, userID).First(&inst).Error; err != nil {
		return nil, errors.New("instância não encontrada")
	}

	// Verifica se já existe config para esta instância (incluindo soft-deleted)
	var existing models.ChatwootConfig
	err := database.DB.Unscoped().Where("instance_id = ?", req.InstanceID).First(&existing).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Se existe mas foi soft-deleted, restaura
	if err == nil && existing.DeletedAt.Valid {
		database.DB.Unscoped().Model(&existing).Update("deleted_at", nil)
		// Continua como update abaixo
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Nova config - verifica quota
		var count int64
		database.DB.Model(&models.ChatwootConfig{}).Where("user_id = ?", userID).Count(&count)
		if int(count) >= user.MaxChatwootConns {
			return nil, fmt.Errorf("limite de conexões Chatwoot atingido (%d/%d)", count, user.MaxChatwootConns)
		}
	}

	// Webhook URL do Chatwoot (aponta de volta para o IMPA HUB)
	chatwootWebhookURL := fmt.Sprintf("%s/chatwoot/webhook/%s", config.AppConfig.BaseURL, req.InstanceID.String())

	inboxID := 0

	// Se auto-create, cria/encontra inbox no Chatwoot
	if req.AutoCreate {
		client := NewChatwootClient(req.URL, req.Token, req.AccountID)
		inbox, err := client.FindOrCreateInbox(req.NameInbox, chatwootWebhookURL)
		if err != nil {
			log.Printf("[IMPA HUB] Aviso: não foi possível criar inbox automaticamente: %v", err)
		} else {
			inboxID = inbox.ID
			log.Printf("[IMPA HUB] Inbox '%s' (ID: %d) configurada para instância %s", req.NameInbox, inbox.ID, inst.EvoInstanceName)
		}
	}

	delimiter := req.SignDelimiter
	if delimiter == "" {
		delimiter = "\n"
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Cria nova config
		cfg := models.ChatwootConfig{
			UserID:              userID,
			InstanceID:          req.InstanceID,
			Enabled:             req.Enabled,
			URL:                 req.URL,
			Token:               req.Token,
			AccountID:           req.AccountID,
			NameInbox:           req.NameInbox,
			InboxID:             inboxID,
			SignMsg:             req.SignMsg,
			SignDelimiter:       delimiter,
			Number:              req.Number,
			ReopenConversation:  req.ReopenConversation,
			ConversationPending: req.ConversationPending,
			MergeBrazilContacts: req.MergeBrazilContacts,
			ImportContacts:      req.ImportContacts,
			ImportMessages:      req.ImportMessages,
			DaysLimitImport:     req.DaysLimitImport,
			AutoCreate:          req.AutoCreate,
			Organization:        req.Organization,
			Logo:                req.Logo,
			GroupsIgnore:        req.GroupsIgnore,
			IgnoreJids:          req.IgnoreJids,
		}

		if err := database.DB.Create(&cfg).Error; err != nil {
			return nil, err
		}

		return &ChatwootConfigResponse{
			ID:                  cfg.ID,
			InstanceID:          cfg.InstanceID,
			InstanceName:        inst.EvoInstanceName,
			Enabled:             cfg.Enabled,
			URL:                 cfg.URL,
			Token:               cfg.Token,
			AccountID:           cfg.AccountID,
			NameInbox:           cfg.NameInbox,
			InboxID:             inboxID,
			SignMsg:             cfg.SignMsg,
			SignDelimiter:       cfg.SignDelimiter,
			ReopenConversation:  cfg.ReopenConversation,
			ConversationPending: cfg.ConversationPending,
			MergeBrazilContacts: cfg.MergeBrazilContacts,
			AutoCreate:          cfg.AutoCreate,
			IsActive:            cfg.Enabled,
			WebhookURL:          chatwootWebhookURL,
			GroupsIgnore:        cfg.GroupsIgnore,
			IgnoreJids:          cfg.IgnoreJids,
		}, nil
	}

	// Atualiza config existente
	updates := map[string]interface{}{
		"enabled":               req.Enabled,
		"url":                   req.URL,
		"token":                 req.Token,
		"account_id":            req.AccountID,
		"name_inbox":            req.NameInbox,
		"sign_msg":              req.SignMsg,
		"sign_delimiter":        delimiter,
		"number":                req.Number,
		"reopen_conversation":   req.ReopenConversation,
		"conversation_pending":  req.ConversationPending,
		"merge_brazil_contacts": req.MergeBrazilContacts,
		"import_contacts":       req.ImportContacts,
		"import_messages":       req.ImportMessages,
		"days_limit_import":     req.DaysLimitImport,
		"auto_create":           req.AutoCreate,
		"organization":          req.Organization,
		"logo":                  req.Logo,
		"groups_ignore":         req.GroupsIgnore,
		"ignore_jids":           req.IgnoreJids,
	}
	if inboxID > 0 {
		updates["inbox_id"] = inboxID
	}

	database.DB.Model(&existing).Updates(updates)

	return &ChatwootConfigResponse{
		ID:                  existing.ID,
		InstanceID:          existing.InstanceID,
		InstanceName:        inst.EvoInstanceName,
		Enabled:             req.Enabled,
		URL:                 req.URL,
		Token:               req.Token,
		AccountID:           req.AccountID,
		NameInbox:           req.NameInbox,
		InboxID:             inboxID,
		SignMsg:             req.SignMsg,
		SignDelimiter:       req.SignDelimiter,
		ReopenConversation:  req.ReopenConversation,
		ConversationPending: req.ConversationPending,
		MergeBrazilContacts: req.MergeBrazilContacts,
		AutoCreate:          req.AutoCreate,
		IsActive:            req.Enabled,
		WebhookURL:          chatwootWebhookURL,
		GroupsIgnore:        req.GroupsIgnore,
		IgnoreJids:          req.IgnoreJids,
	}, nil
}

func GetChatwootConfig(userID, instanceID uuid.UUID) (*ChatwootConfigResponse, error) {
	var cfg models.ChatwootConfig
	if err := database.DB.Where("instance_id = ? AND user_id = ?", instanceID, userID).First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("configuração Chatwoot não encontrada")
		}
		return nil, err
	}

	var inst models.Instance
	database.DB.First(&inst, cfg.InstanceID)

	webhookURL := fmt.Sprintf("%s/chatwoot/webhook/%s", config.AppConfig.BaseURL, instanceID.String())

	return &ChatwootConfigResponse{
		ID:                  cfg.ID,
		InstanceID:          cfg.InstanceID,
		InstanceName:        inst.EvoInstanceName,
		Enabled:             cfg.Enabled,
		URL:                 cfg.URL,
		Token:               cfg.Token,
		AccountID:           cfg.AccountID,
		NameInbox:           cfg.NameInbox,
		InboxID:             cfg.InboxID,
		SignMsg:             cfg.SignMsg,
		SignDelimiter:       cfg.SignDelimiter,
		ReopenConversation:  cfg.ReopenConversation,
		ConversationPending: cfg.ConversationPending,
		MergeBrazilContacts: cfg.MergeBrazilContacts,
		AutoCreate:          cfg.AutoCreate,
		IsActive:            cfg.Enabled,
		WebhookURL:          webhookURL,
		GroupsIgnore:        cfg.GroupsIgnore,
		IgnoreJids:          cfg.IgnoreJids,
	}, nil
}

func ListChatwootConfigs(userID uuid.UUID) ([]ChatwootConfigResponse, error) {
	var configs []models.ChatwootConfig
	if err := database.DB.Where("user_id = ?", userID).Find(&configs).Error; err != nil {
		return nil, err
	}

	var result []ChatwootConfigResponse
	for _, cfg := range configs {
		var inst models.Instance
		database.DB.First(&inst, cfg.InstanceID)

		webhookURL := fmt.Sprintf("%s/chatwoot/webhook/%s", config.AppConfig.BaseURL, cfg.InstanceID.String())

		result = append(result, ChatwootConfigResponse{
			ID:                  cfg.ID,
			InstanceID:          cfg.InstanceID,
			InstanceName:        inst.EvoInstanceName,
			Enabled:             cfg.Enabled,
			URL:                 cfg.URL,
			Token:               cfg.Token,
			AccountID:           cfg.AccountID,
			NameInbox:           cfg.NameInbox,
			InboxID:             cfg.InboxID,
			SignMsg:             cfg.SignMsg,
			SignDelimiter:       cfg.SignDelimiter,
			ReopenConversation:  cfg.ReopenConversation,
			ConversationPending: cfg.ConversationPending,
			MergeBrazilContacts: cfg.MergeBrazilContacts,
			AutoCreate:          cfg.AutoCreate,
			IsActive:            cfg.Enabled,
			WebhookURL:          webhookURL,
			GroupsIgnore:        cfg.GroupsIgnore,
			IgnoreJids:          cfg.IgnoreJids,
		})
	}

	return result, nil
}

func UpdateChatwootConfig(userID, instanceID uuid.UUID, req UpdateChatwootRequest) error {
	var cfg models.ChatwootConfig
	if err := database.DB.Where("instance_id = ? AND user_id = ?", instanceID, userID).First(&cfg).Error; err != nil {
		return errors.New("configuração Chatwoot não encontrada")
	}

	updates := make(map[string]interface{})
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Token != nil {
		updates["token"] = *req.Token
	}
	if req.AccountID != nil {
		updates["account_id"] = *req.AccountID
	}
	if req.NameInbox != nil {
		updates["name_inbox"] = *req.NameInbox
	}
	if req.SignMsg != nil {
		updates["sign_msg"] = *req.SignMsg
	}
	if req.SignDelimiter != nil {
		updates["sign_delimiter"] = *req.SignDelimiter
	}
	if req.Number != nil {
		updates["number"] = *req.Number
	}
	if req.ReopenConversation != nil {
		updates["reopen_conversation"] = *req.ReopenConversation
	}
	if req.ConversationPending != nil {
		updates["conversation_pending"] = *req.ConversationPending
	}
	if req.MergeBrazilContacts != nil {
		updates["merge_brazil_contacts"] = *req.MergeBrazilContacts
	}
	if req.AutoCreate != nil {
		updates["auto_create"] = *req.AutoCreate
	}
	if req.Organization != nil {
		updates["organization"] = *req.Organization
	}
	if req.Logo != nil {
		updates["logo"] = *req.Logo
	}

	return database.DB.Model(&cfg).Updates(updates).Error
}

func DeleteChatwootConfig(userID, instanceID uuid.UUID) error {
	var cfg models.ChatwootConfig
	if err := database.DB.Where("instance_id = ? AND user_id = ?", instanceID, userID).First(&cfg).Error; err != nil {
		return errors.New("configuração Chatwoot não encontrada")
	}

	// Limpa conversas e mapeamentos
	database.DB.Where("chatwoot_config_id = ?", cfg.ID).Delete(&models.ChatwootConversation{})
	database.DB.Where("chatwoot_config_id = ?", cfg.ID).Delete(&models.ChatwootMessageMap{})

	return database.DB.Delete(&cfg).Error
}

// GetConfigByInstanceID retorna a config Chatwoot pelo ID da instância (uso interno)
func GetConfigByInstanceID(instanceID uuid.UUID) (*models.ChatwootConfig, error) {
	var cfg models.ChatwootConfig
	if err := database.DB.Where("instance_id = ?", instanceID).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}
