package typebot

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/models"
	"gorm.io/gorm"
)

// ==================== Request/Response Types ====================

type CreateTypebotRequest struct {
	InstanceID         uuid.UUID              `json:"instance_id" binding:"required"`
	Enabled            bool                   `json:"enabled"`
	Description        string                 `json:"description"`
	URL                string                 `json:"url" binding:"required"`
	Typebot            string                 `json:"typebot" binding:"required"`
	TriggerType        string                 `json:"trigger_type"`
	TriggerOperator    string                 `json:"trigger_operator"`
	TriggerValue       string                 `json:"trigger_value"`
	Expire             int                    `json:"expire"`
	KeywordFinish      string                 `json:"keyword_finish"`
	DelayMessage       int                    `json:"delay_message"`
	UnknownMessage     string                 `json:"unknown_message"`
	ListeningFromMe    bool                   `json:"listening_from_me"`
	StopBotFromMe      bool                   `json:"stop_bot_from_me"`
	KeepOpen           bool                   `json:"keep_open"`
	DebounceTime       int                    `json:"debounce_time"`
	PrefilledVariables map[string]interface{} `json:"prefilled_variables,omitempty"`
	IgnoreJids         []string               `json:"ignore_jids,omitempty"`
}

type UpdateTypebotRequest struct {
	Enabled            *bool                  `json:"enabled,omitempty"`
	Description        *string                `json:"description,omitempty"`
	URL                *string                `json:"url,omitempty"`
	Typebot            *string                `json:"typebot,omitempty"`
	TriggerType        *string                `json:"trigger_type,omitempty"`
	TriggerOperator    *string                `json:"trigger_operator,omitempty"`
	TriggerValue       *string                `json:"trigger_value,omitempty"`
	Expire             *int                   `json:"expire,omitempty"`
	KeywordFinish      *string                `json:"keyword_finish,omitempty"`
	DelayMessage       *int                   `json:"delay_message,omitempty"`
	UnknownMessage     *string                `json:"unknown_message,omitempty"`
	ListeningFromMe    *bool                  `json:"listening_from_me,omitempty"`
	StopBotFromMe      *bool                  `json:"stop_bot_from_me,omitempty"`
	KeepOpen           *bool                  `json:"keep_open,omitempty"`
	DebounceTime       *int                   `json:"debounce_time,omitempty"`
	PrefilledVariables map[string]interface{} `json:"prefilled_variables,omitempty"`
	IgnoreJids         []string               `json:"ignore_jids,omitempty"`
}

type TypebotConfigResponse struct {
	ID                 uuid.UUID              `json:"id"`
	InstanceID         uuid.UUID              `json:"instance_id"`
	InstanceName       string                 `json:"instance_name"`
	Enabled            bool                   `json:"enabled"`
	Description        string                 `json:"description"`
	URL                string                 `json:"url"`
	Typebot            string                 `json:"typebot"`
	TriggerType        string                 `json:"trigger_type"`
	TriggerOperator    string                 `json:"trigger_operator"`
	TriggerValue       string                 `json:"trigger_value"`
	Expire             int                    `json:"expire"`
	KeywordFinish      string                 `json:"keyword_finish"`
	DelayMessage       int                    `json:"delay_message"`
	UnknownMessage     string                 `json:"unknown_message"`
	ListeningFromMe    bool                   `json:"listening_from_me"`
	StopBotFromMe      bool                   `json:"stop_bot_from_me"`
	KeepOpen           bool                   `json:"keep_open"`
	DebounceTime       int                    `json:"debounce_time"`
	PrefilledVariables map[string]interface{} `json:"prefilled_variables,omitempty"`
	IgnoreJids         []string               `json:"ignore_jids,omitempty"`
}

type SetSettingsRequest struct {
	Expire            *int       `json:"expire,omitempty"`
	KeywordFinish     *string    `json:"keyword_finish,omitempty"`
	DelayMessage      *int       `json:"delay_message,omitempty"`
	UnknownMessage    *string    `json:"unknown_message,omitempty"`
	ListeningFromMe   *bool      `json:"listening_from_me,omitempty"`
	StopBotFromMe     *bool      `json:"stop_bot_from_me,omitempty"`
	KeepOpen          *bool      `json:"keep_open,omitempty"`
	DebounceTime      *int       `json:"debounce_time,omitempty"`
	TypebotIDFallback *uuid.UUID `json:"typebot_id_fallback,omitempty"`
	IgnoreJids        []string   `json:"ignore_jids,omitempty"`
}

type SettingsResponse struct {
	ID                uuid.UUID             `json:"id"`
	InstanceID        uuid.UUID             `json:"instance_id"`
	Expire            int                   `json:"expire"`
	KeywordFinish     string                `json:"keyword_finish"`
	DelayMessage      int                   `json:"delay_message"`
	UnknownMessage    string                `json:"unknown_message"`
	ListeningFromMe   bool                  `json:"listening_from_me"`
	StopBotFromMe     bool                  `json:"stop_bot_from_me"`
	KeepOpen          bool                  `json:"keep_open"`
	DebounceTime      int                   `json:"debounce_time"`
	TypebotIDFallback *uuid.UUID            `json:"typebot_id_fallback,omitempty"`
	IgnoreJids        []string              `json:"ignore_jids,omitempty"`
	Fallback          *TypebotConfigResponse `json:"fallback,omitempty"`
}

type IgnoreJidRequest struct {
	Action string `json:"action" binding:"required"` // "add" or "remove"
	JID    string `json:"jid" binding:"required"`
}

type StartTypebotRequest struct {
	RemoteJID          string                 `json:"remote_jid" binding:"required"`
	TypebotConfigID    *uuid.UUID             `json:"typebot_config_id,omitempty"`
	URL                string                 `json:"url,omitempty"`
	Typebot            string                 `json:"typebot,omitempty"`
	PrefilledVariables map[string]interface{} `json:"prefilled_variables,omitempty"`
}

type ChangeStatusRequest struct {
	RemoteJID string `json:"remote_jid" binding:"required"`
	Status    string `json:"status" binding:"required"` // opened|closed|paused|delete
}

type SessionResponse struct {
	ID              uuid.UUID `json:"id"`
	RemoteJID       string    `json:"remote_jid"`
	PushName        string    `json:"push_name"`
	SessionID       string    `json:"session_id"`
	Status          string    `json:"status"`
	AwaitUser       bool      `json:"await_user"`
	TypebotConfigID uuid.UUID `json:"typebot_config_id"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

// ==================== CRUD: TypebotConfig (multi-bot) ====================

func CreateTypebotConfig(userID uuid.UUID, req CreateTypebotRequest) (*TypebotConfigResponse, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("usuário não encontrado")
	}
	if !user.CanUseTypebot {
		return nil, errors.New("você não tem permissão para usar Typebot")
	}

	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", req.InstanceID, userID).First(&inst).Error; err != nil {
		return nil, errors.New("instância não encontrada")
	}

	// Verifica quota total
	var count int64
	database.DB.Model(&models.TypebotConfig{}).Where("user_id = ?", userID).Count(&count)
	if int(count) >= user.MaxTypebotConns {
		return nil, fmt.Errorf("limite de configurações Typebot atingido (%d/%d)", count, user.MaxTypebotConns)
	}

	// trigger "all" permite apenas 1 por instância
	if req.TriggerType == "all" {
		var existingAll int64
		database.DB.Model(&models.TypebotConfig{}).
			Where("instance_id = ? AND trigger_type = ? AND enabled = ?", req.InstanceID, "all", true).
			Count(&existingAll)
		if existingAll > 0 {
			return nil, errors.New("já existe um Typebot com trigger 'all' para esta instância")
		}
	}

	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = "keyword"
	}
	triggerOp := req.TriggerOperator
	if triggerOp == "" {
		triggerOp = "contains"
	}
	delayMsg := req.DelayMessage
	if delayMsg == 0 {
		delayMsg = 1000
	}

	prefilledJSON := ""
	if req.PrefilledVariables != nil {
		data, _ := json.Marshal(req.PrefilledVariables)
		prefilledJSON = string(data)
	}
	ignoreJSON := ""
	if req.IgnoreJids != nil {
		data, _ := json.Marshal(req.IgnoreJids)
		ignoreJSON = string(data)
	}

	cfg := models.TypebotConfig{
		UserID:             userID,
		InstanceID:         req.InstanceID,
		Enabled:            req.Enabled,
		Description:        req.Description,
		URL:                req.URL,
		Typebot:            req.Typebot,
		TriggerType:        triggerType,
		TriggerOperator:    triggerOp,
		TriggerValue:       req.TriggerValue,
		Expire:             req.Expire,
		KeywordFinish:      req.KeywordFinish,
		DelayMessage:       delayMsg,
		UnknownMessage:     req.UnknownMessage,
		ListeningFromMe:    req.ListeningFromMe,
		StopBotFromMe:      req.StopBotFromMe,
		KeepOpen:           req.KeepOpen,
		DebounceTime:       req.DebounceTime,
		PrefilledVariables: prefilledJSON,
		IgnoreJids:         ignoreJSON,
	}

	if err := database.DB.Create(&cfg).Error; err != nil {
		return nil, err
	}
	return toConfigResponse(&cfg, inst.EvoInstanceName), nil
}

func FindTypebotConfigs(userID uuid.UUID) ([]TypebotConfigResponse, error) {
	var configs []models.TypebotConfig
	if err := database.DB.Where("user_id = ?", userID).Find(&configs).Error; err != nil {
		return nil, err
	}
	var results []TypebotConfigResponse
	for _, cfg := range configs {
		var inst models.Instance
		database.DB.First(&inst, cfg.InstanceID)
		results = append(results, *toConfigResponse(&cfg, inst.EvoInstanceName))
	}
	return results, nil
}

func FindTypebotConfigsByInstance(userID, instanceID uuid.UUID) ([]TypebotConfigResponse, error) {
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		return nil, errors.New("instância não encontrada")
	}
	var configs []models.TypebotConfig
	if err := database.DB.Where("instance_id = ? AND user_id = ?", instanceID, userID).Find(&configs).Error; err != nil {
		return nil, err
	}
	var results []TypebotConfigResponse
	for _, cfg := range configs {
		results = append(results, *toConfigResponse(&cfg, inst.EvoInstanceName))
	}
	return results, nil
}

func FetchTypebotConfig(userID, typebotID uuid.UUID) (*TypebotConfigResponse, error) {
	var cfg models.TypebotConfig
	if err := database.DB.Where("id = ? AND user_id = ?", typebotID, userID).First(&cfg).Error; err != nil {
		return nil, errors.New("configuração Typebot não encontrada")
	}
	var inst models.Instance
	database.DB.First(&inst, cfg.InstanceID)
	return toConfigResponse(&cfg, inst.EvoInstanceName), nil
}

func UpdateTypebotConfig(userID, typebotID uuid.UUID, req UpdateTypebotRequest) error {
	var cfg models.TypebotConfig
	if err := database.DB.Where("id = ? AND user_id = ?", typebotID, userID).First(&cfg).Error; err != nil {
		return errors.New("configuração Typebot não encontrada")
	}

	if req.TriggerType != nil && *req.TriggerType == "all" && cfg.TriggerType != "all" {
		var existingAll int64
		database.DB.Model(&models.TypebotConfig{}).
			Where("instance_id = ? AND trigger_type = ? AND enabled = ? AND id != ?", cfg.InstanceID, "all", true, cfg.ID).
			Count(&existingAll)
		if existingAll > 0 {
			return errors.New("já existe um Typebot com trigger 'all' para esta instância")
		}
	}

	updates := make(map[string]interface{})
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Typebot != nil {
		updates["typebot"] = *req.Typebot
	}
	if req.TriggerType != nil {
		updates["trigger_type"] = *req.TriggerType
	}
	if req.TriggerOperator != nil {
		updates["trigger_operator"] = *req.TriggerOperator
	}
	if req.TriggerValue != nil {
		updates["trigger_value"] = *req.TriggerValue
	}
	if req.Expire != nil {
		updates["expire"] = *req.Expire
	}
	if req.KeywordFinish != nil {
		updates["keyword_finish"] = *req.KeywordFinish
	}
	if req.DelayMessage != nil {
		updates["delay_message"] = *req.DelayMessage
	}
	if req.UnknownMessage != nil {
		updates["unknown_message"] = *req.UnknownMessage
	}
	if req.ListeningFromMe != nil {
		updates["listening_from_me"] = *req.ListeningFromMe
	}
	if req.StopBotFromMe != nil {
		updates["stop_bot_from_me"] = *req.StopBotFromMe
	}
	if req.KeepOpen != nil {
		updates["keep_open"] = *req.KeepOpen
	}
	if req.DebounceTime != nil {
		updates["debounce_time"] = *req.DebounceTime
	}
	if req.PrefilledVariables != nil {
		data, _ := json.Marshal(req.PrefilledVariables)
		updates["prefilled_variables"] = string(data)
	}
	if req.IgnoreJids != nil {
		data, _ := json.Marshal(req.IgnoreJids)
		updates["ignore_jids"] = string(data)
	}
	return database.DB.Model(&cfg).Updates(updates).Error
}

func DeleteTypebotConfig(userID, typebotID uuid.UUID) error {
	var cfg models.TypebotConfig
	if err := database.DB.Where("id = ? AND user_id = ?", typebotID, userID).First(&cfg).Error; err != nil {
		return errors.New("configuração Typebot não encontrada")
	}
	database.DB.Where("typebot_config_id = ?", cfg.ID).Delete(&models.TypebotSession{})
	database.DB.Model(&models.TypebotSetting{}).
		Where("typebot_id_fallback = ?", cfg.ID).
		Update("typebot_id_fallback", nil)
	return database.DB.Delete(&cfg).Error
}

// ==================== CRUD: TypebotSetting ====================

func SetSettings(userID, instanceID uuid.UUID, req SetSettingsRequest) (*SettingsResponse, error) {
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		return nil, errors.New("instância não encontrada")
	}

	if req.TypebotIDFallback != nil {
		var fallback models.TypebotConfig
		if err := database.DB.Where("id = ? AND instance_id = ?", req.TypebotIDFallback, instanceID).First(&fallback).Error; err != nil {
			return nil, errors.New("Typebot fallback não encontrado para esta instância")
		}
	}

	var setting models.TypebotSetting
	err := database.DB.Where("instance_id = ?", instanceID).First(&setting).Error

	ignoreJSON := ""
	if req.IgnoreJids != nil {
		data, _ := json.Marshal(req.IgnoreJids)
		ignoreJSON = string(data)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = models.TypebotSetting{
			InstanceID:        instanceID,
			TypebotIDFallback: req.TypebotIDFallback,
			IgnoreJids:        ignoreJSON,
			DelayMessage:      1000,
		}
		if req.Expire != nil {
			setting.Expire = *req.Expire
		}
		if req.KeywordFinish != nil {
			setting.KeywordFinish = *req.KeywordFinish
		}
		if req.DelayMessage != nil {
			setting.DelayMessage = *req.DelayMessage
		}
		if req.UnknownMessage != nil {
			setting.UnknownMessage = *req.UnknownMessage
		}
		if req.ListeningFromMe != nil {
			setting.ListeningFromMe = *req.ListeningFromMe
		}
		if req.StopBotFromMe != nil {
			setting.StopBotFromMe = *req.StopBotFromMe
		}
		if req.KeepOpen != nil {
			setting.KeepOpen = *req.KeepOpen
		}
		if req.DebounceTime != nil {
			setting.DebounceTime = *req.DebounceTime
		}
		if err := database.DB.Create(&setting).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		updates := make(map[string]interface{})
		if req.Expire != nil {
			updates["expire"] = *req.Expire
		}
		if req.KeywordFinish != nil {
			updates["keyword_finish"] = *req.KeywordFinish
		}
		if req.DelayMessage != nil {
			updates["delay_message"] = *req.DelayMessage
		}
		if req.UnknownMessage != nil {
			updates["unknown_message"] = *req.UnknownMessage
		}
		if req.ListeningFromMe != nil {
			updates["listening_from_me"] = *req.ListeningFromMe
		}
		if req.StopBotFromMe != nil {
			updates["stop_bot_from_me"] = *req.StopBotFromMe
		}
		if req.KeepOpen != nil {
			updates["keep_open"] = *req.KeepOpen
		}
		if req.DebounceTime != nil {
			updates["debounce_time"] = *req.DebounceTime
		}
		if req.TypebotIDFallback != nil {
			updates["typebot_id_fallback"] = *req.TypebotIDFallback
		}
		if ignoreJSON != "" {
			updates["ignore_jids"] = ignoreJSON
		}
		database.DB.Model(&setting).Updates(updates)
		database.DB.First(&setting, setting.ID)
	}

	return toSettingsResponse(&setting, userID), nil
}

func FetchSettings(userID, instanceID uuid.UUID) (*SettingsResponse, error) {
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		return nil, errors.New("instância não encontrada")
	}

	var setting models.TypebotSetting
	if err := database.DB.Where("instance_id = ?", instanceID).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &SettingsResponse{InstanceID: instanceID, DelayMessage: 1000}, nil
		}
		return nil, err
	}
	return toSettingsResponse(&setting, userID), nil
}

func IgnoreJid(userID, instanceID uuid.UUID, req IgnoreJidRequest) error {
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		return errors.New("instância não encontrada")
	}

	var setting models.TypebotSetting
	err := database.DB.Where("instance_id = ?", instanceID).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if req.Action == "add" {
			jids, _ := json.Marshal([]string{req.JID})
			setting = models.TypebotSetting{InstanceID: instanceID, DelayMessage: 1000, IgnoreJids: string(jids)}
			return database.DB.Create(&setting).Error
		}
		return nil
	}
	if err != nil {
		return err
	}

	var jids []string
	if setting.IgnoreJids != "" {
		json.Unmarshal([]byte(setting.IgnoreJids), &jids)
	}

	switch req.Action {
	case "add":
		for _, j := range jids {
			if j == req.JID {
				return nil
			}
		}
		jids = append(jids, req.JID)
	case "remove":
		filtered := make([]string, 0, len(jids))
		for _, j := range jids {
			if j != req.JID {
				filtered = append(filtered, j)
			}
		}
		jids = filtered
	default:
		return errors.New("action deve ser 'add' ou 'remove'")
	}

	data, _ := json.Marshal(jids)
	return database.DB.Model(&setting).Update("ignore_jids", string(data)).Error
}

// ==================== Session Operations ====================

func GetSessions(userID, instanceID uuid.UUID) ([]SessionResponse, error) {
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		return nil, errors.New("instância não encontrada")
	}

	var configIDs []uuid.UUID
	database.DB.Model(&models.TypebotConfig{}).Where("instance_id = ?", instanceID).Pluck("id", &configIDs)
	if len(configIDs) == 0 {
		return []SessionResponse{}, nil
	}

	var sessions []models.TypebotSession
	database.DB.Where("typebot_config_id IN ?", configIDs).Find(&sessions)

	var results []SessionResponse
	for _, s := range sessions {
		results = append(results, toSessionResponse(&s))
	}
	return results, nil
}

func ChangeSessionStatus(userID, instanceID uuid.UUID, req ChangeStatusRequest) error {
	var inst models.Instance
	if err := database.DB.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		return errors.New("instância não encontrada")
	}

	var configIDs []uuid.UUID
	database.DB.Model(&models.TypebotConfig{}).Where("instance_id = ?", instanceID).Pluck("id", &configIDs)
	if len(configIDs) == 0 {
		return errors.New("nenhuma configuração Typebot encontrada")
	}

	if req.Status == "delete" {
		return database.DB.Where("typebot_config_id IN ? AND remote_jid = ?", configIDs, req.RemoteJID).
			Delete(&models.TypebotSession{}).Error
	}
	return database.DB.Model(&models.TypebotSession{}).
		Where("typebot_config_id IN ? AND remote_jid = ?", configIDs, req.RemoteJID).
		Update("status", req.Status).Error
}

// ==================== Helpers ====================

func toConfigResponse(cfg *models.TypebotConfig, instanceName string) *TypebotConfigResponse {
	var prefilled map[string]interface{}
	if cfg.PrefilledVariables != "" {
		json.Unmarshal([]byte(cfg.PrefilledVariables), &prefilled)
	}
	var ignoreJids []string
	if cfg.IgnoreJids != "" {
		json.Unmarshal([]byte(cfg.IgnoreJids), &ignoreJids)
	}
	return &TypebotConfigResponse{
		ID: cfg.ID, InstanceID: cfg.InstanceID, InstanceName: instanceName,
		Enabled: cfg.Enabled, Description: cfg.Description,
		URL: cfg.URL, Typebot: cfg.Typebot,
		TriggerType: cfg.TriggerType, TriggerOperator: cfg.TriggerOperator, TriggerValue: cfg.TriggerValue,
		Expire: cfg.Expire, KeywordFinish: cfg.KeywordFinish, DelayMessage: cfg.DelayMessage,
		UnknownMessage: cfg.UnknownMessage, ListeningFromMe: cfg.ListeningFromMe,
		StopBotFromMe: cfg.StopBotFromMe, KeepOpen: cfg.KeepOpen, DebounceTime: cfg.DebounceTime,
		PrefilledVariables: prefilled, IgnoreJids: ignoreJids,
	}
}

func toSettingsResponse(s *models.TypebotSetting, userID uuid.UUID) *SettingsResponse {
	var ignoreJids []string
	if s.IgnoreJids != "" {
		json.Unmarshal([]byte(s.IgnoreJids), &ignoreJids)
	}
	resp := &SettingsResponse{
		ID: s.ID, InstanceID: s.InstanceID,
		Expire: s.Expire, KeywordFinish: s.KeywordFinish, DelayMessage: s.DelayMessage,
		UnknownMessage: s.UnknownMessage, ListeningFromMe: s.ListeningFromMe,
		StopBotFromMe: s.StopBotFromMe, KeepOpen: s.KeepOpen, DebounceTime: s.DebounceTime,
		TypebotIDFallback: s.TypebotIDFallback, IgnoreJids: ignoreJids,
	}
	if s.TypebotIDFallback != nil {
		var fallback models.TypebotConfig
		if database.DB.First(&fallback, s.TypebotIDFallback).Error == nil {
			var inst models.Instance
			database.DB.First(&inst, fallback.InstanceID)
			resp.Fallback = toConfigResponse(&fallback, inst.EvoInstanceName)
		}
	}
	return resp
}

func toSessionResponse(s *models.TypebotSession) SessionResponse {
	return SessionResponse{
		ID: s.ID, RemoteJID: s.RemoteJID, PushName: s.PushName,
		SessionID: s.SessionID, Status: string(s.Status), AwaitUser: s.AwaitUser,
		TypebotConfigID: s.TypebotConfigID,
		CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
