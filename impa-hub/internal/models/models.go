package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== Base Model ====================

type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// ==================== User (Multi-tenant) ====================

type UserRole string

const (
	RoleSuperAdmin UserRole = "superadmin"
	RoleAdmin      UserRole = "admin"
	RoleUser       UserRole = "user"
)

type User struct {
	BaseModel
	Name         string   `gorm:"not null" json:"name"`
	Email        string   `gorm:"uniqueIndex;not null" json:"email"`
	Password     string   `gorm:"not null" json:"-"`
	Role         UserRole `gorm:"type:varchar(20);default:'user'" json:"role"`
	Active       bool     `gorm:"default:true" json:"is_active"`

	// Quotas (definidas pelo SuperAdmin)
	MaxInstances         int  `gorm:"default:5" json:"max_instances"`
	MaxChatwootConns     int  `gorm:"default:5" json:"max_chatwoot_conns"`
	MaxTypebotConns      int  `gorm:"default:5" json:"max_typebot_conns"`
	MaxEvoServers        int  `gorm:"default:3" json:"max_evo_servers"`
	CanUseChatwoot       bool `gorm:"default:true" json:"can_use_chatwoot"`
	CanUseTypebot        bool `gorm:"default:true" json:"can_use_typebot"`

	// Relações
	EvoServers       []EvoServer       `gorm:"foreignKey:UserID" json:"evo_servers,omitempty"`
	Instances        []Instance        `gorm:"foreignKey:UserID" json:"instances,omitempty"`
	ChatwootConfigs  []ChatwootConfig  `gorm:"foreignKey:UserID" json:"chatwoot_configs,omitempty"`
	TypebotConfigs   []TypebotConfig   `gorm:"foreignKey:UserID" json:"typebot_configs,omitempty"`
}

// ==================== EvoServer (Conexão com Evolution GO) ====================

type EvoServer struct {
	BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name      string    `gorm:"not null" json:"name"`
	URL       string    `gorm:"not null" json:"base_url"`    // Ex: https://api.example.com
	APIKey    string    `gorm:"not null" json:"-"`            // GLOBAL_API_KEY do server
	Active    bool      `gorm:"default:true" json:"is_active"`

	// Relações
	User      User       `gorm:"foreignKey:UserID" json:"-"`
	Instances []Instance `gorm:"foreignKey:EvoServerID" json:"instances,omitempty"`
}

// ==================== Instance (Instância WhatsApp no Evolution GO) ====================

type InstanceStatus string

const (
	StatusDisconnected InstanceStatus = "disconnected"
	StatusConnecting   InstanceStatus = "connecting"
	StatusConnected    InstanceStatus = "connected"
	StatusQRCode       InstanceStatus = "qrcode"
)

type Instance struct {
	BaseModel
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	EvoServerID uuid.UUID      `gorm:"type:uuid;not null;index" json:"server_id"`
	
	// Dados da instância no Evolution GO
	EvoInstanceID   string `gorm:"index" json:"evo_instance_id"`
	EvoInstanceName string `gorm:"not null" json:"instance_name"`
	EvoToken        string `json:"-"`
	
	// Estado
	Status    InstanceStatus `gorm:"type:varchar(20);default:'disconnected'" json:"connection_status"`
	Phone     string         `json:"phone,omitempty"`
	PushName  string         `json:"push_name,omitempty"`
	
	// Webhook (apontando para IMPA HUB)
	WebhookConfigured bool `gorm:"default:false" json:"webhook_configured"`

	// Relações
	User            User            `gorm:"foreignKey:UserID" json:"-"`
	EvoServer       EvoServer       `gorm:"foreignKey:EvoServerID" json:"-"`
	ChatwootConfig  *ChatwootConfig `gorm:"foreignKey:InstanceID" json:"chatwoot_config,omitempty"`
	TypebotConfigs  []TypebotConfig `gorm:"foreignKey:InstanceID" json:"typebot_configs,omitempty"`
	TypebotSetting  *TypebotSetting `gorm:"foreignKey:InstanceID" json:"typebot_setting,omitempty"`
}

// ==================== ChatwootConfig ====================

type ChatwootConfig struct {
	BaseModel
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	InstanceID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"instance_id"`
	
	// Conexão com Chatwoot
	Enabled    bool   `gorm:"default:true" json:"enabled"`
	URL        string `gorm:"not null" json:"url"`          // URL do Chatwoot
	Token      string `gorm:"not null" json:"-"`             // API access token
	AccountID  string `gorm:"not null" json:"account_id"`     // Account ID no Chatwoot

	// Inbox
	NameInbox  string `gorm:"not null" json:"name_inbox"`
	InboxID    int    `json:"inbox_id,omitempty"`

	// Configurações
	SignMsg             bool   `gorm:"default:true" json:"sign_msg"`
	SignDelimiter       string `gorm:"default:'\\n'" json:"sign_delimiter"`
	ReopenConversation  bool   `gorm:"default:false" json:"reopen_conversation"`
	ConversationPending bool   `gorm:"default:false" json:"conversation_pending"`
	MergeBrazilContacts bool   `gorm:"default:true" json:"merge_brazil_contacts"`
	ImportContacts      bool   `gorm:"default:false" json:"import_contacts"`
	ImportMessages      bool   `gorm:"default:false" json:"import_messages"`
	DaysLimitImport     int    `gorm:"default:30" json:"days_limit_import"`
	AutoCreate          bool   `gorm:"default:true" json:"auto_create"`
	Organization        string `json:"organization,omitempty"`
	Logo                string `json:"logo,omitempty"`

	// Grupos
	GroupsIgnore bool `gorm:"default:true" json:"groups_ignore"` // true = ignora grupos (padrão), false = processa grupos

	// JIDs a ignorar (JSON array)
	IgnoreJids string `gorm:"type:text" json:"ignore_jids,omitempty"` // JSON array of strings

	// Número WhatsApp (registrado no Chatwoot)
	Number string `json:"number,omitempty"`

	// Relações
	User     User     `gorm:"foreignKey:UserID" json:"-"`
	Instance Instance `gorm:"foreignKey:InstanceID" json:"-"`
}

// ==================== WebhookLog ====================

type WebhookLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	InstanceID uuid.UUID `gorm:"type:uuid;index" json:"instance_id"`
	Event      string    `gorm:"not null;index" json:"event"`
	Direction  string    `gorm:"type:varchar(10);not null" json:"direction"` // "incoming" or "outgoing"
	Payload    string    `gorm:"type:text" json:"payload"`
	Status     string    `gorm:"type:varchar(20)" json:"status"` // "success", "error"
	Error      string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

func (w *WebhookLog) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// ==================== ChatwootConversation Cache ====================

type ChatwootConversation struct {
	BaseModel
	ChatwootConfigID uuid.UUID `gorm:"type:uuid;not null;index" json:"chatwoot_config_id"`
	RemoteJID        string    `gorm:"column:remote_jid;not null;index" json:"remote_jid"`
	ContactID        int       `gorm:"not null" json:"contact_id"`
	ConversationID   int       `gorm:"not null" json:"conversation_id"`
	InboxID          int       `json:"inbox_id"`
	SourceID         string    `json:"source_id,omitempty"`
}

// ==================== TypebotConfig ====================

type TypebotConfig struct {
	BaseModel
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	InstanceID uuid.UUID `gorm:"type:uuid;not null;index" json:"instance_id"`

	// Conexão com Typebot
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Description string `gorm:"type:varchar(255)" json:"description,omitempty"`
	URL         string `gorm:"not null" json:"url"`     // URL do servidor Typebot
	Typebot     string `gorm:"not null" json:"typebot"` // ID público do Typebot

	// Gatilho (trigger)
	TriggerType     string `gorm:"type:varchar(20);default:'keyword'" json:"trigger_type"` // all|keyword|none|advanced
	TriggerOperator string `gorm:"type:varchar(20);default:'contains'" json:"trigger_operator"` // contains|equals|startsWith|endsWith|regex
	TriggerValue    string `json:"trigger_value,omitempty"`

	// Configurações de comportamento
	Expire          int    `gorm:"default:0" json:"expire"`                // Minutos até expirar sessão (0=nunca)
	KeywordFinish   string `json:"keyword_finish,omitempty"`                // Palavra para encerrar
	DelayMessage    int    `gorm:"default:1000" json:"delay_message"`       // ms entre mensagens
	UnknownMessage  string `json:"unknown_message,omitempty"`               // Resposta se não entende
	ListeningFromMe bool   `gorm:"default:false" json:"listening_from_me"`   // Processa msgs enviadas por mim
	StopBotFromMe   bool   `gorm:"default:false" json:"stop_bot_from_me"`     // Para bot se eu enviar msg
	KeepOpen        bool   `gorm:"default:false" json:"keep_open"`          // Mantém sessão closed após terminar
	DebounceTime    int    `gorm:"default:0" json:"debounce_time"`          // Segundos para agrupar msgs

	// Variáveis pré-preenchidas (JSON)
	PrefilledVariables string `gorm:"type:text" json:"prefilled_variables,omitempty"` // JSON string

	// JIDs a ignorar (JSON array)
	IgnoreJids string `gorm:"type:text" json:"ignore_jids,omitempty"` // JSON array of strings

	// Relações
	User     User     `gorm:"foreignKey:UserID" json:"-"`
	Instance Instance `gorm:"foreignKey:InstanceID" json:"-"`
}

// ==================== TypebotSetting (Global per Instance) ====================

type TypebotSetting struct {
	BaseModel
	InstanceID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"instance_id"`

	// Configurações padrão (herdadas se o bot não definir)
	Expire          int    `gorm:"default:0" json:"expire"`
	KeywordFinish   string `json:"keyword_finish,omitempty"`
	DelayMessage    int    `gorm:"default:1000" json:"delay_message"`
	UnknownMessage  string `json:"unknown_message,omitempty"`
	ListeningFromMe bool   `gorm:"default:false" json:"listening_from_me"`
	StopBotFromMe   bool   `gorm:"default:false" json:"stop_bot_from_me"`
	KeepOpen        bool   `gorm:"default:false" json:"keep_open"`
	DebounceTime    int    `gorm:"default:0" json:"debounce_time"`

	// Bot fallback
	TypebotIDFallback *uuid.UUID `gorm:"type:uuid" json:"typebot_id_fallback,omitempty"`

	// JIDs a ignorar globalmente (JSON array)
	IgnoreJids string `gorm:"type:text" json:"ignore_jids,omitempty"` // JSON array of strings

	// Relações
	Instance Instance      `gorm:"foreignKey:InstanceID" json:"-"`
	Fallback *TypebotConfig `gorm:"foreignKey:TypebotIDFallback" json:"fallback,omitempty"`
}

// ==================== TypebotSession ====================

type TypebotSessionStatus string

const (
	TypebotSessionOpened TypebotSessionStatus = "opened"
	TypebotSessionClosed TypebotSessionStatus = "closed"
	TypebotSessionPaused TypebotSessionStatus = "paused"
)

type TypebotSession struct {
	BaseModel
	TypebotConfigID uuid.UUID            `gorm:"type:uuid;not null;index" json:"typebot_config_id"`
	RemoteJID       string               `gorm:"not null;index" json:"remote_jid"`
	PushName        string               `json:"push_name,omitempty"`
	SessionID       string               `gorm:"not null" json:"session_id"` // formato: randomId-typebotSessionId
	Status          TypebotSessionStatus `gorm:"type:varchar(20);default:'opened'" json:"status"`
	AwaitUser       bool                 `gorm:"default:false" json:"await_user"`
	Parameters      string               `gorm:"type:text" json:"parameters,omitempty"` // JSON string

	// Relações
	TypebotConfig TypebotConfig `gorm:"foreignKey:TypebotConfigID" json:"-"`
}

// ==================== ChatwootMessage Mapping ====================

type ChatwootMessageMap struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ChatwootConfigID  uuid.UUID `gorm:"type:uuid;not null;index" json:"chatwoot_config_id"`
	WhatsAppMessageID string    `gorm:"not null;index" json:"whatsapp_message_id"`
	ChatwootMessageID int       `gorm:"not null;index" json:"chatwoot_message_id"`
	ConversationID    int       `gorm:"not null" json:"conversation_id"`
	CreatedAt         time.Time `json:"created_at"`
}

func (c *ChatwootMessageMap) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
