package chatwoot

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/evoclient"
	"github.com/impa-hub/internal/models"
)

// ==================== Anti-duplicação Chatwoot → WhatsApp → Chatwoot ====================
// Quando uma msg é enviada via Chatwoot, guardamos um hash por 30s.
// Se processSentMessage receber o mesmo conteúdo para o mesmo JID nesse período, ignora.

var (
	chatwootSentCache   = make(map[string]time.Time)
	chatwootSentCacheMu sync.Mutex
)

func chatwootSentKey(cfgID uuid.UUID, phone, content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%s:%s:%x", cfgID, phone, h[:8])
}

func markSentFromChatwoot(cfgID uuid.UUID, phone, content string) {
	chatwootSentCacheMu.Lock()
	defer chatwootSentCacheMu.Unlock()
	key := chatwootSentKey(cfgID, phone, content)
	chatwootSentCache[key] = time.Now()
	// Limpa entradas antigas (> 60s)
	for k, t := range chatwootSentCache {
		if time.Since(t) > 60*time.Second {
			delete(chatwootSentCache, k)
		}
	}
}

func wasSentFromChatwoot(cfgID uuid.UUID, phone, content string) bool {
	chatwootSentCacheMu.Lock()
	defer chatwootSentCacheMu.Unlock()
	key := chatwootSentKey(cfgID, phone, content)
	if t, ok := chatwootSentCache[key]; ok {
		if time.Since(t) < 30*time.Second {
			delete(chatwootSentCache, key) // usa só 1 vez
			return true
		}
		delete(chatwootSentCache, key)
	}
	return false
}

// ==================== Anti-duplicação de MÍDIA Chatwoot → WhatsApp → Chatwoot ====================
// Contador de mídias pendentes por JID. Quando Chatwoot envia N anexos, incrementamos N.
// Cada SendMessage de mídia que volta decrementa 1. Enquanto > 0, ignora.

var (
	chatwootMediaPending   = make(map[string]int)
	chatwootMediaPendingTs = make(map[string]time.Time)
	chatwootMediaPendingMu sync.Mutex
)

func markMediaSentFromChatwoot(cfgID uuid.UUID, phone string, count int) {
	chatwootMediaPendingMu.Lock()
	defer chatwootMediaPendingMu.Unlock()
	key := fmt.Sprintf("%s:%s", cfgID, phone)
	chatwootMediaPending[key] += count
	chatwootMediaPendingTs[key] = time.Now()
	// Limpa entradas antigas (> 60s)
	for k, t := range chatwootMediaPendingTs {
		if time.Since(t) > 60*time.Second {
			delete(chatwootMediaPending, k)
			delete(chatwootMediaPendingTs, k)
		}
	}
}

func wasMediaSentFromChatwoot(cfgID uuid.UUID, phone string) bool {
	chatwootMediaPendingMu.Lock()
	defer chatwootMediaPendingMu.Unlock()
	key := fmt.Sprintf("%s:%s", cfgID, phone)
	if count, ok := chatwootMediaPending[key]; ok && count > 0 {
		if t, hasTs := chatwootMediaPendingTs[key]; hasTs && time.Since(t) < 30*time.Second {
			chatwootMediaPending[key]--
			if chatwootMediaPending[key] <= 0 {
				delete(chatwootMediaPending, key)
				delete(chatwootMediaPendingTs, key)
			}
			return true
		}
		delete(chatwootMediaPending, key)
		delete(chatwootMediaPendingTs, key)
	}
	return false
}

// ==================== Tipos de Evento do Evolution GO ====================

type EvoWebhookPayload struct {
	Event         string          `json:"event"`
	InstanceToken string          `json:"instanceToken"`
	InstanceID    string          `json:"instanceId"`
	InstanceName  string          `json:"instanceName"`
	Data          json.RawMessage `json:"data"`
}

type EvoMessageData struct {
	MessageType string          `json:"messageType"`
	Message     json.RawMessage `json:"Message"`
	Info        EvoMessageInfo  `json:"Info"`
	Quoted      json.RawMessage `json:"quoted,omitempty"`
	IsQuoted    bool            `json:"isQuoted"`
	MediaURL    string          `json:"mediaUrl,omitempty"`
	Base64      string          `json:"base64,omitempty"`
}

// extractMediaFromMessage extrai mediaUrl e base64 de dentro do campo Message
// pois o Evolution GO coloca esses campos nested dentro de Message, não no top-level
func (m *EvoMessageData) extractMediaFromMessage() {
	if m.MediaURL != "" || m.Base64 != "" {
		return // já tem no top-level
	}
	if len(m.Message) == 0 {
		return
	}
	var msgMap map[string]json.RawMessage
	if err := json.Unmarshal(m.Message, &msgMap); err != nil {
		return
	}
	if raw, ok := msgMap["mediaUrl"]; ok {
		var url string
		if json.Unmarshal(raw, &url) == nil {
			m.MediaURL = url
		}
	}
	if raw, ok := msgMap["base64"]; ok {
		var b64 string
		if json.Unmarshal(raw, &b64) == nil {
			m.Base64 = b64
		}
	}
}

type EvoMessageInfo struct {
	ID        string `json:"ID"`
	Chat      string `json:"Chat"`
	Sender    string `json:"Sender"`
	Timestamp string `json:"Timestamp"`
	IsFromMe  bool   `json:"IsFromMe"`
	PushName  string `json:"PushName,omitempty"`
}

type EvoConnectionData struct {
	Status   string `json:"status"`
	JID      string `json:"jid"`
	PushName string `json:"pushName"`
	Reason   string `json:"reason,omitempty"`
}

// ==================== Tipos de Evento do Chatwoot ====================

type ChatwootWebhookPayload struct {
	Event             string                `json:"event"`
	MessageType       string                `json:"message_type"`
	Content           string                `json:"content"`
	ContentType       string                `json:"content_type"`
	ContentAttributes json.RawMessage       `json:"content_attributes,omitempty"`
	Private           bool                  `json:"private"`
	Conversation      ChatwootConvPayload   `json:"conversation"`
	Sender            ChatwootSenderPayload `json:"sender"`
	Account           ChatwootAccountData   `json:"account"`
	Inbox             ChatwootInboxData     `json:"inbox"`
	Attachments       []ChatwootAttachment  `json:"attachments,omitempty"`
	ID                int                   `json:"id"`
}

type ChatwootConvPayload struct {
	ID           int    `json:"id"`
	Status       string `json:"status"`
	ContactInbox struct {
		SourceID string `json:"source_id"`
	} `json:"contact_inbox"`
	Meta struct {
		Sender struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			PhoneNumber string `json:"phone_number"`
			Identifier  string `json:"identifier"`
		} `json:"sender"`
	} `json:"meta"`
}

type ChatwootSenderPayload struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "user" (agente) ou "contact"
}

type ChatwootAccountData struct {
	ID int `json:"id"`
}

type ChatwootInboxData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ChatwootAttachment struct {
	ID       int    `json:"id"`
	FileType string `json:"file_type"` // "image", "audio", "video", "file"
	DataURL  string `json:"data_url"`
	ThumbURL string `json:"thumb_url,omitempty"`
	FileSize int    `json:"file_size,omitempty"`
}

type ChatwootContentAttributes struct {
	Deleted bool `json:"deleted,omitempty"`
}

// ==================== Processador Webhook EVO -> Chatwoot ====================

// ProcessEvoWebhook processa um evento recebido do Evolution GO e encaminha para o Chatwoot
func ProcessEvoWebhook(instanceID uuid.UUID, payload EvoWebhookPayload) {
	// Busca configuração Chatwoot para esta instância
	cfg, err := GetConfigByInstanceID(instanceID)
	if err != nil {
		// Sem config Chatwoot - ignora silenciosamente
		return
	}

	if !cfg.Enabled {
		return
	}

	switch payload.Event {
	case "Message":
		processIncomingMessage(cfg, payload)
	case "SendMessage":
		processSentMessage(cfg, payload)
	case "Connected", "PairSuccess":
		processConnection(instanceID, payload)
	case "LoggedOut", "Disconnected":
		processDisconnection(instanceID, payload)
	case "Receipt":
		// Pode ser usado para marcar como lida
		processReceipt(cfg, payload)
	default:
		log.Printf("[IMPA HUB] Evento EVO não processado para Chatwoot: %s", payload.Event)
	}
}

func processIncomingMessage(cfg *models.ChatwootConfig, payload EvoWebhookPayload) {
	var msgData EvoMessageData
	if err := json.Unmarshal(payload.Data, &msgData); err != nil {
		log.Printf("[IMPA HUB] Erro ao parsear mensagem EVO: %v", err)
		return
	}
	msgData.extractMediaFromMessage()

	// Mensagens enviadas por nós (do app ou outro dispositivo) - redireciona para processSentMessage
	if msgData.Info.IsFromMe {
		processSentMessage(cfg, payload)
		return
	}

	// Ignora status/broadcast
	if strings.HasSuffix(msgData.Info.Chat, "@broadcast") || msgData.Info.Chat == "status@broadcast" {
		return
	}

	// Verifica ignore JIDs configurados
	if isIgnoredJID(cfg.IgnoreJids, msgData.Info.Chat) {
		log.Printf("[IMPA HUB] Ignorando mensagem de JID configurado: %s", msgData.Info.Chat)
		return
	}

	// Verifica se é mensagem de grupo
	if isGroup(msgData.Info.Chat) {
		if !cfg.GroupsIgnore {
			// Grupos habilitados: processa como mensagem de grupo
			processGroupMessage(cfg, msgData)
			return
		}
		// Grupos desabilitados (padrão): ignora
		log.Printf("[IMPA HUB] Ignorando mensagem de grupo: %s", msgData.Info.Chat)
		return
	}

	// Extrai número do remetente
	remoteJID := msgData.Info.Sender
	if remoteJID == "" {
		remoteJID = msgData.Info.Chat
	}

	phoneNumber := cleanJID(remoteJID)
	contactName := msgData.Info.PushName
	if contactName == "" {
		contactName = phoneNumber
	}

	client := NewChatwootClient(cfg.URL, cfg.Token, cfg.AccountID)

	// 1. Encontra ou cria contato
	contact, err := findOrCreateContact(client, cfg, phoneNumber, contactName, remoteJID)
	if err != nil {
		log.Printf("[IMPA HUB] Erro ao criar/buscar contato no Chatwoot: %v", err)
		return
	}

	// 2. Encontra ou cria conversa
	conversation, err := findOrCreateConversation(client, cfg, contact, remoteJID)
	if err != nil {
		log.Printf("[IMPA HUB] Erro ao criar/buscar conversa no Chatwoot: %v", err)
		return
	}

	// 3. Extrai conteúdo da mensagem
	content := extractMessageContent(msgData)

	// Detecta se é mensagem de mídia (mesmo sem base64/mediaUrl disponível)
	isMediaMessage := isMediaType(msgData)

	// 4. Envia para o Chatwoot
	var cwMsg *Message

	if msgData.MediaURL != "" || msgData.Base64 != "" {
		// Mensagem com mídia disponível
		log.Printf("[IMPA HUB] Mídia detectada: type=%s, mediaUrl=%v, base64=%v", msgData.MessageType, msgData.MediaURL != "", msgData.Base64 != "")
		cwMsg, err = sendMediaToChatwoot(client, conversation.ConversationID, content, msgData, "incoming")
	} else if isMediaMessage {
		// Mensagem de mídia mas sem arquivo (S3/Minio falhou na EVO)
		log.Printf("[IMPA HUB] Mídia sem arquivo disponível (S3 falhou?): type=%s, id=%s", msgData.MessageType, msgData.Info.ID)
		if content == "" {
			content = mediaFallbackText(msgData)
		}

		cwMsg, err = client.SendMessage(conversation.ConversationID, MessagePayload{
			Content:     content,
			MessageType: "incoming",
		})
	} else {
		// Mensagem de texto
		if content == "" {
			return // Sem conteúdo para enviar
		}

		cwMsg, err = client.SendMessage(conversation.ConversationID, MessagePayload{
			Content:     content,
			MessageType: "incoming",
		})
	}

	if err != nil {
		log.Printf("[IMPA HUB] Erro ao enviar mensagem para Chatwoot: %v", err)
		return
	}

	// 5. Salva mapeamento de mensagem
	if cwMsg != nil {
		mapping := models.ChatwootMessageMap{
			ChatwootConfigID:  cfg.ID,
			WhatsAppMessageID: msgData.Info.ID,
			ChatwootMessageID: cwMsg.ID,
			ConversationID:    conversation.ConversationID,
		}
		database.DB.Create(&mapping)
	}

	// Log
	logWebhook(cfg.InstanceID, "Message", "incoming", "success", "")
}

func processSentMessage(cfg *models.ChatwootConfig, payload EvoWebhookPayload) {
	var msgData EvoMessageData
	if err := json.Unmarshal(payload.Data, &msgData); err != nil {
		return
	}
	msgData.extractMediaFromMessage()

	// Mensagem que nós enviamos (via WhatsApp direto) - registra no Chatwoot como outgoing
	if !msgData.Info.IsFromMe {
		return
	}

	// Verifica mensagens de grupo
	if isGroup(msgData.Info.Chat) {
		if cfg.GroupsIgnore {
			return // Grupos desabilitados
		}
		// Grupos habilitados: processa como mensagem de grupo enviada
		processGroupSentMessage(cfg, msgData)
		return
	}

	remoteJID := msgData.Info.Chat

	// Anti-duplicação: se essa msg foi enviada via Chatwoot, não registrar de novo
	content := extractMessageContent(msgData)
	phoneForCheck := cleanJID(remoteJID)
	// Verifica anti-dup para mídia (contador) e para texto (hash de conteúdo)
	hasMedia := msgData.MediaURL != "" || msgData.Base64 != "" || isMediaType(msgData)
	if hasMedia && wasMediaSentFromChatwoot(cfg.ID, phoneForCheck) {
		log.Printf("[IMPA HUB] Anti-dup: ignorando SendMessage de mídia que veio do Chatwoot (JID=%s)", remoteJID)
		return
	}
	if wasSentFromChatwoot(cfg.ID, phoneForCheck, content) {
		log.Printf("[IMPA HUB] Anti-dup: ignorando SendMessage que veio do Chatwoot (JID=%s)", remoteJID)
		return
	}
	phoneNumber := cleanJID(remoteJID)

	client := NewChatwootClient(cfg.URL, cfg.Token, cfg.AccountID)

	// Busca conversa existente
	var conv models.ChatwootConversation
	if err := database.DB.Where("chatwoot_config_id = ? AND remote_jid = ?", cfg.ID, remoteJID).First(&conv).Error; err != nil {
		// Sem conversa, tenta com phone number
		if err2 := database.DB.Where("chatwoot_config_id = ? AND remote_jid LIKE ?", cfg.ID, "%"+phoneNumber+"%").First(&conv).Error; err2 != nil {
			// Cria contato e conversa para registrar mensagens enviadas do app
			log.Printf("[IMPA HUB] Criando contato/conversa para mensagem enviada do app (JID=%s)", remoteJID)
			contact, err := findOrCreateContact(client, cfg, phoneNumber, phoneNumber, remoteJID)
			if err != nil {
				log.Printf("[IMPA HUB] Erro ao criar contato para mensagem enviada: %v", err)
				return
			}
			conversation, err := findOrCreateConversation(client, cfg, contact, remoteJID)
			if err != nil {
				log.Printf("[IMPA HUB] Erro ao criar conversa para mensagem enviada: %v", err)
				return
			}
			conv = *conversation
		}
	}

	var cwMsg *Message
	var err error

	if msgData.MediaURL != "" || msgData.Base64 != "" {
		cwMsg, err = sendMediaToChatwoot(client, conv.ConversationID, content, msgData, "outgoing")
	} else if isMediaType(msgData) {
		// Mídia sem arquivo (S3 falhou)
		if content == "" {
			content = mediaFallbackText(msgData)
		}
		cwMsg, err = client.SendMessage(conv.ConversationID, MessagePayload{
			Content:     content,
			MessageType: "outgoing",
		})
	} else {
		if content == "" {
			return
		}
		cwMsg, err = client.SendMessage(conv.ConversationID, MessagePayload{
			Content:     content,
			MessageType: "outgoing",
		})
	}

	if err != nil {
		log.Printf("[IMPA HUB] Erro ao registrar mensagem enviada no Chatwoot: %v", err)
		return
	}

	if cwMsg != nil {
		mapping := models.ChatwootMessageMap{
			ChatwootConfigID:  cfg.ID,
			WhatsAppMessageID: msgData.Info.ID,
			ChatwootMessageID: cwMsg.ID,
			ConversationID:    conv.ConversationID,
		}
		database.DB.Create(&mapping)
	}

	log.Printf("[IMPA HUB] Mensagem enviada do app registrada no Chatwoot (JID=%s, conv=%d)", remoteJID, conv.ConversationID)
}

func processConnection(instanceID uuid.UUID, payload EvoWebhookPayload) {
	var connData EvoConnectionData
	if err := json.Unmarshal(payload.Data, &connData); err != nil {
		return
	}

	updates := map[string]interface{}{
		"status": models.StatusConnected,
	}
	if connData.JID != "" {
		updates["phone"] = connData.JID
	}
	if connData.PushName != "" {
		updates["push_name"] = connData.PushName
	}

	database.DB.Model(&models.Instance{}).Where("id = ?", instanceID).Updates(updates)
	log.Printf("[IMPA HUB] Instância %s conectada (JID: %s)", instanceID, connData.JID)
}

func processDisconnection(instanceID uuid.UUID, payload EvoWebhookPayload) {
	database.DB.Model(&models.Instance{}).Where("id = ?", instanceID).Update("status", models.StatusDisconnected)
	log.Printf("[IMPA HUB] Instância %s desconectada", instanceID)
}

func processReceipt(cfg *models.ChatwootConfig, payload EvoWebhookPayload) {
	// Pode ser implementado para marcar mensagens como lidas no Chatwoot
	// Por enquanto, apenas loga
}

// processGroupMessage processa mensagens de grupo no Chatwoot
// Cria um contato para o grupo e prefixa as mensagens com o participante
func processGroupMessage(cfg *models.ChatwootConfig, msgData EvoMessageData) {
	groupJID := msgData.Info.Chat

	// Extrai o participante (quem enviou a mensagem no grupo)
	participantJID := msgData.Info.Sender
	if participantJID == "" || participantJID == groupJID {
		log.Printf("[IMPA HUB] Mensagem de grupo sem participante identificável: %s", groupJID)
		return
	}

	participantPhone := cleanJID(participantJID)
	participantName := msgData.Info.PushName
	if participantName == "" {
		participantName = participantPhone
	}

	// Nome do contato do grupo no Chatwoot - busca nome real via API
	groupID := cleanJID(groupJID)
	groupName := fetchGroupName(cfg, groupJID)
	if groupName == "" {
		log.Printf("\n\n!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		log.Printf("[IMPA HUB] GRUPO IGNORADO - NÃO FOI POSSÍVEL OBTER O NOME DO GRUPO")
		log.Printf("[IMPA HUB] GroupJID: %s | Mensagem NÃO será enviada ao Chatwoot", groupJID)
		log.Printf("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
		return
	}

	client := NewChatwootClient(cfg.URL, cfg.Token, cfg.AccountID)

	// 1. Encontra ou cria contato do GRUPO
	groupContact, err := findOrCreateContact(client, cfg, groupID, groupName, groupJID)
	if err != nil {
		log.Printf("[IMPA HUB] Erro ao criar/buscar contato de grupo no Chatwoot: %v", err)
		return
	}

	// 2. Encontra ou cria conversa do GRUPO
	conversation, err := findOrCreateConversation(client, cfg, groupContact, groupJID)
	if err != nil {
		log.Printf("[IMPA HUB] Erro ao criar/buscar conversa de grupo no Chatwoot: %v", err)
		return
	}

	// 3. Extrai conteúdo da mensagem
	content := extractMessageContent(msgData)
	isMediaMessage := isMediaType(msgData)

	// 4. Prefixa com identificação do participante (como a Evolution API faz)
	prefix := fmt.Sprintf("**+%s - %s:**\n\n", participantPhone, participantName)

	// 5. Envia para o Chatwoot
	var cwMsg *Message

	if msgData.MediaURL != "" || msgData.Base64 != "" {
		// Mídia com prefixo do participante
		if content != "" {
			content = prefix + content
		} else {
			content = prefix
		}
		cwMsg, err = sendMediaToChatwoot(client, conversation.ConversationID, content, msgData, "incoming")
	} else if isMediaMessage {
		if content == "" {
			content = mediaFallbackText(msgData)
		}
		content = prefix + content
		cwMsg, err = client.SendMessage(conversation.ConversationID, MessagePayload{
			Content:     content,
			MessageType: "incoming",
		})
	} else {
		if content == "" {
			return
		}
		content = prefix + content
		cwMsg, err = client.SendMessage(conversation.ConversationID, MessagePayload{
			Content:     content,
			MessageType: "incoming",
		})
	}

	if err != nil {
		log.Printf("[IMPA HUB] Erro ao enviar mensagem de grupo para Chatwoot: %v", err)
		return
	}

	// 6. Salva mapeamento
	if cwMsg != nil {
		mapping := models.ChatwootMessageMap{
			ChatwootConfigID:  cfg.ID,
			WhatsAppMessageID: msgData.Info.ID,
			ChatwootMessageID: cwMsg.ID,
			ConversationID:    conversation.ConversationID,
		}
		database.DB.Create(&mapping)
	}

	log.Printf("[IMPA HUB] Mensagem de grupo processada: grupo=%s participante=%s", groupJID, participantJID)
}

// processGroupSentMessage processa mensagens enviadas por nós em grupos
func processGroupSentMessage(cfg *models.ChatwootConfig, msgData EvoMessageData) {
	groupJID := msgData.Info.Chat
	groupID := cleanJID(groupJID)
	groupName := fetchGroupName(cfg, groupJID)
	if groupName == "" {
		log.Printf("\n\n!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		log.Printf("[IMPA HUB] GRUPO ENVIADO IGNORADO - NÃO FOI POSSÍVEL OBTER O NOME DO GRUPO")
		log.Printf("[IMPA HUB] GroupJID: %s | Mensagem enviada NÃO será registrada no Chatwoot", groupJID)
		log.Printf("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
		return
	}

	client := NewChatwootClient(cfg.URL, cfg.Token, cfg.AccountID)

	// Anti-duplicação: se essa msg foi enviada via Chatwoot, não registrar de novo
	content := extractMessageContent(msgData)
	phoneForCheck := cleanJID(groupJID)
	// Verifica anti-dup para mídia (contador) e para texto (hash de conteúdo)
	hasMedia := msgData.MediaURL != "" || msgData.Base64 != "" || isMediaType(msgData)
	if hasMedia && wasMediaSentFromChatwoot(cfg.ID, phoneForCheck) {
		log.Printf("[IMPA HUB] Anti-dup: ignorando SendMessage de mídia de grupo que veio do Chatwoot (JID=%s)", groupJID)
		return
	}
	if wasSentFromChatwoot(cfg.ID, phoneForCheck, content) {
		log.Printf("[IMPA HUB] Anti-dup: ignorando SendMessage de grupo que veio do Chatwoot (JID=%s)", groupJID)
		return
	}

	// 1. Encontra ou cria contato do GRUPO
	groupContact, err := findOrCreateContact(client, cfg, groupID, groupName, groupJID)
	if err != nil {
		log.Printf("[IMPA HUB] Erro ao criar/buscar contato de grupo para mensagem enviada: %v", err)
		return
	}

	// 2. Encontra ou cria conversa do GRUPO
	conversation, err := findOrCreateConversation(client, cfg, groupContact, groupJID)
	if err != nil {
		log.Printf("[IMPA HUB] Erro ao criar/buscar conversa de grupo para mensagem enviada: %v", err)
		return
	}

	// 3. Envia para Chatwoot como outgoing
	var cwMsg *Message

	if msgData.MediaURL != "" || msgData.Base64 != "" {
		cwMsg, err = sendMediaToChatwoot(client, conversation.ConversationID, content, msgData, "outgoing")
	} else if isMediaType(msgData) {
		if content == "" {
			content = mediaFallbackText(msgData)
		}
		cwMsg, err = client.SendMessage(conversation.ConversationID, MessagePayload{
			Content:     content,
			MessageType: "outgoing",
		})
	} else {
		if content == "" {
			return
		}
		cwMsg, err = client.SendMessage(conversation.ConversationID, MessagePayload{
			Content:     content,
			MessageType: "outgoing",
		})
	}

	if err != nil {
		log.Printf("[IMPA HUB] Erro ao registrar mensagem de grupo enviada no Chatwoot: %v", err)
		return
	}

	if cwMsg != nil {
		mapping := models.ChatwootMessageMap{
			ChatwootConfigID:  cfg.ID,
			WhatsAppMessageID: msgData.Info.ID,
			ChatwootMessageID: cwMsg.ID,
			ConversationID:    conversation.ConversationID,
		}
		database.DB.Create(&mapping)
	}

	log.Printf("[IMPA HUB] Mensagem de grupo enviada processada: grupo=%s", groupJID)
}

// ==================== Processador Webhook Chatwoot -> EVO GO ====================

// ProcessChatwootWebhook processa um evento recebido do Chatwoot e encaminha para o WhatsApp
func ProcessChatwootWebhook(instanceID uuid.UUID, payload ChatwootWebhookPayload) error {
	// Busca config
	cfg, err := GetConfigByInstanceID(instanceID)
	if err != nil {
		return fmt.Errorf("configuração Chatwoot não encontrada para instância %s", instanceID)
	}

	if !cfg.Enabled {
		return nil
	}

	switch payload.Event {
	case "message_created":
		return processChatwootMessage(cfg, instanceID, payload)
	case "message_updated":
		return processChatwootMessageUpdate(cfg, instanceID, payload)
	case "conversation_status_changed":
		return processChatwootStatusChange(cfg, payload)
	default:
		log.Printf("[IMPA HUB] Evento Chatwoot não processado: %s", payload.Event)
	}

	return nil
}

func processChatwootMessage(cfg *models.ChatwootConfig, instanceID uuid.UUID, payload ChatwootWebhookPayload) error {
	// Ignora mensagens incoming (do contato) - já processamos via EVO
	if payload.MessageType != "outgoing" {
		return nil
	}

	// Ignora mensagens privadas
	if payload.Private {
		return nil
	}

	// Ignora se o sender é um contato (não é agente)
	if payload.Sender.Type == "contact" {
		return nil
	}

	// Anti-loop: ignora mensagens que foram criadas pelo próprio IMPA HUB
	// (processSentMessage cria a msg no Chatwoot e salva na ChatwootMessageMap)
	if payload.ID > 0 {
		var existingMap models.ChatwootMessageMap
		if database.DB.Where("chatwoot_config_id = ? AND chatwoot_message_id = ?",
			cfg.ID, payload.ID).First(&existingMap).Error == nil {
			log.Printf("[IMPA HUB] Anti-loop: ignorando mensagem %d (criada pelo próprio IMPA HUB)", payload.ID)
			return nil
		}
	}

	// Obtém número do destinatário
	phoneNumber := payload.Conversation.Meta.Sender.PhoneNumber
	if phoneNumber == "" {
		phoneNumber = payload.Conversation.Meta.Sender.Identifier
	}

	if phoneNumber == "" {
		// Tenta buscar nosso mapeamento interno
		var conv models.ChatwootConversation
		if err := database.DB.Where("chatwoot_config_id = ? AND conversation_id = ?", cfg.ID, payload.Conversation.ID).First(&conv).Error; err != nil {
			return fmt.Errorf("não foi possível determinar o destinatário")
		}
		phoneNumber = cleanJID(conv.RemoteJID)
	}

	phoneNumber = cleanPhoneNumber(phoneNumber)

	if phoneNumber == "" || phoneNumber == "123456" {
		return nil // Ignora contato bot
	}

	// Normaliza phone para anti-dup (mesmo formato que processSentMessage/processGroupSentMessage usam cleanJID)
	phoneForAntiDup := cleanJID(phoneNumber)

	// Busca instância e servidor
	var inst models.Instance
	if err := database.DB.First(&inst, instanceID).Error; err != nil {
		return fmt.Errorf("instância não encontrada")
	}

	var srv models.EvoServer
	if err := database.DB.First(&srv, inst.EvoServerID).Error; err != nil {
		return fmt.Errorf("servidor não encontrado")
	}

	evoClient := evoclient.New(srv.URL, srv.APIKey)

	// Processa anexos
	if len(payload.Attachments) > 0 {
		// Assina caption se configurado
		caption := payload.Content
		if cfg.SignMsg && payload.Sender.Name != "" && caption != "" {
			delimiter := strings.ReplaceAll(cfg.SignDelimiter, `\n`, "\n")
			if delimiter == "" {
				delimiter = "\n"
			}
			caption = fmt.Sprintf("*%s:*%s%s", payload.Sender.Name, delimiter, caption)
		}
		// Marca no cache anti-dup: contador de mídias + caption
		markMediaSentFromChatwoot(cfg.ID, phoneForAntiDup, len(payload.Attachments))
		if caption != "" {
			markSentFromChatwoot(cfg.ID, phoneForAntiDup, caption)
		}

		// Cache para evitar duplicatas no mesmo webhook
		processedAttachments := make(map[int]bool)

		for _, att := range payload.Attachments {
			// Verifica se já processou este attachment
			if processedAttachments[att.ID] {
				log.Printf("[IMPA HUB] DEBUG: Ignorando attachment duplicado %d", att.ID)
				continue
			}
			processedAttachments[att.ID] = true
			mediaType := mapFileTypeToMediaType(att.FileType)
			filename := extractAttachmentFilename(att)
			if filename == "" {
				filename = fmt.Sprintf("attachment_%d", att.ID)
			}

			// Tenta extrair URL direta do arquivo (sem redirect)
			directURL := att.DataURL
			if strings.Contains(att.DataURL, "/redirect/") {
				// Tenta obter URL real baixando o conteúdo do redirect
				log.Printf("[IMPA HUB] DEBUG: Tentando extrair URL direta do redirect %s", att.DataURL)

				httpClient := &http.Client{Timeout: 10 * time.Second}
				resp, err := httpClient.Get(att.DataURL)
				if err == nil && resp.StatusCode == 200 {
					defer resp.Body.Close()
					body, _ := io.ReadAll(resp.Body)

					// Procura pela URL real no conteúdo HTML do redirect
					bodyStr := string(body)
					if strings.Contains(bodyStr, "href=") {
						// Extrai URL do atributo href
						start := strings.Index(bodyStr, "href=") + 6
						end := strings.Index(bodyStr[start:], "\"")
						if end > start {
							directURL = bodyStr[start:end]
							log.Printf("[IMPA HUB] DEBUG: URL direta extraída: %s", directURL)
						}
					}
				}
			}

			log.Printf("[IMPA HUB] DEBUG: Enviando mídia - attachment_id=%d, filename=%s, mediaType=%s, url_length=%d",
				att.ID, filename, mediaType, len(directURL))

			// Envia URL (direta se conseguiu extrair, senão original)
			resp, err := evoClient.SendMedia(inst.EvoToken, evoclient.SendMediaRequest{
				Number:   phoneNumber,
				Type:     mediaType,
				Caption:  caption,
				Filename: filename,
				URL:      directURL,
			})

			if err != nil {
				log.Printf("[IMPA HUB] ERRO: Falha ao enviar mídia %s (attachment_id=%d): %v", filename, att.ID, err)
				// Log detalhado para debug
				if strings.Contains(err.Error(), "ffmpeg") {
					log.Printf("[IMPA HUB] ERRO: Problema de conversão FFmpeg - possivelmente formato não suportado")
				}
				if strings.Contains(err.Error(), "pipe:0") {
					log.Printf("[IMPA HUB] ERRO: Problema com pipe de dados - URL pode estar corrompida")
				}
			} else {
				log.Printf("[IMPA HUB] SUCESSO: Mídia enviada %s (attachment_id=%d)", filename, att.ID)
				// Log da resposta do Evolution GO para análise
				if len(resp) > 0 {
					var respData map[string]interface{}
					if json.Unmarshal(resp, &respData) == nil {
						if msg, ok := respData["message"]; ok {
							log.Printf("[IMPA HUB] DEBUG: Resposta Evolution GO: %s", msg)
						}
					}
				}
			}
		}
		logWebhook(instanceID, "message_created", "outgoing", "success", "")
		return nil
	}

	// Envia texto
	if payload.Content != "" {
		textToSend := payload.Content

		// Assina mensagem com nome do agente do Chatwoot (fluxo Chatwoot → WhatsApp)
		if cfg.SignMsg && payload.Sender.Name != "" {
			delimiter := strings.ReplaceAll(cfg.SignDelimiter, `\n`, "\n")
			if delimiter == "" {
				delimiter = "\n"
			}
			textToSend = fmt.Sprintf("*%s:*%s%s", payload.Sender.Name, delimiter, textToSend)
		}

		// Marca no cache anti-dup antes de enviar
		markSentFromChatwoot(cfg.ID, phoneForAntiDup, textToSend)

		_, err := evoClient.SendText(inst.EvoToken, evoclient.SendTextRequest{
			Number: phoneNumber,
			Text:   textToSend,
		})
		if err != nil {
			logWebhook(instanceID, "message_created", "outgoing", "error", err.Error())
			return err
		}
	}

	logWebhook(instanceID, "message_created", "outgoing", "success", "")
	return nil
}

func processChatwootMessageUpdate(cfg *models.ChatwootConfig, instanceID uuid.UUID, payload ChatwootWebhookPayload) error {
	// Verifica se é deleção
	if payload.ContentAttributes != nil {
		var attrs ChatwootContentAttributes
		if err := json.Unmarshal(payload.ContentAttributes, &attrs); err == nil && attrs.Deleted {
			// Mensagem deletada no Chatwoot - podemos deletar no WhatsApp se quisermos
			log.Printf("[IMPA HUB] Mensagem deletada no Chatwoot (conv: %d, msg: %d)", payload.Conversation.ID, payload.ID)
		}
	}
	return nil
}

func processChatwootStatusChange(cfg *models.ChatwootConfig, payload ChatwootWebhookPayload) error {
	log.Printf("[IMPA HUB] Conversa %d mudou de status para: %s", payload.Conversation.ID, payload.Conversation.Status)
	return nil
}

// ==================== Helpers ====================

// Cache de nomes de grupo para evitar chamadas repetidas à API
var (
	groupNameCache   = make(map[string]string)
	groupNameCacheMu sync.Mutex
)

// fetchGroupName busca o nome real do grupo via Evolution GO API
// Retorna string vazia se não conseguir obter o nome (NÃO envia fallback)
func fetchGroupName(cfg *models.ChatwootConfig, groupJID string) string {
	// Verifica cache
	groupNameCacheMu.Lock()
	if name, ok := groupNameCache[groupJID]; ok {
		groupNameCacheMu.Unlock()
		log.Printf("[IMPA HUB] fetchGroupName: cache hit para %s → %q", groupJID, name)
		return name
	}
	groupNameCacheMu.Unlock()

	// Busca instância
	var inst models.Instance
	if err := database.DB.Where("id = ?", cfg.InstanceID).First(&inst).Error; err != nil {
		log.Printf("\n\n========================================")
		log.Printf("[IMPA HUB] ERRO CRÍTICO: fetchGroupName - instância não encontrada")
		log.Printf("[IMPA HUB] InstanceID: %s | GroupJID: %s | Erro: %v", cfg.InstanceID, groupJID, err)
		log.Printf("========================================\n")
		return ""
	}

	// Busca servidor
	var srv models.EvoServer
	if err := database.DB.Where("id = ?", inst.EvoServerID).First(&srv).Error; err != nil {
		log.Printf("\n\n========================================")
		log.Printf("[IMPA HUB] ERRO CRÍTICO: fetchGroupName - servidor não encontrado")
		log.Printf("[IMPA HUB] ServerID: %s | GroupJID: %s | Erro: %v", inst.EvoServerID, groupJID, err)
		log.Printf("========================================\n")
		return ""
	}

	// Chama a API do Evolution GO
	evoClient := evoclient.New(srv.URL, srv.APIKey)
	log.Printf("[IMPA HUB] fetchGroupName: POST %s/group/info groupJid=%s token=%s...", srv.URL, groupJID, inst.EvoToken[:8])
	info, err := evoClient.GetGroupInfo(inst.EvoToken, groupJID)
	if err != nil {
		log.Printf("\n\n========================================")
		log.Printf("[IMPA HUB] ERRO CRÍTICO: fetchGroupName - API retornou erro")
		log.Printf("[IMPA HUB] URL: %s | GroupJID: %s | Erro: %v", srv.URL, groupJID, err)
		log.Printf("========================================\n")
		return ""
	}

	groupName := info.Name
	log.Printf("[IMPA HUB] fetchGroupName: API retornou Name=%q para %s", groupName, groupJID)

	if groupName == "" {
		log.Printf("\n\n========================================")
		log.Printf("[IMPA HUB] ERRO CRÍTICO: fetchGroupName - API retornou nome VAZIO")
		log.Printf("[IMPA HUB] URL: %s | GroupJID: %s | Response: %+v", srv.URL, groupJID, info)
		log.Printf("========================================\n")
		return ""
	}

	// Salva no cache
	groupNameCacheMu.Lock()
	groupNameCache[groupJID] = groupName
	groupNameCacheMu.Unlock()

	log.Printf("[IMPA HUB] Nome do grupo obtido com sucesso: %s → %q", groupJID, groupName)
	return groupName
}

func findOrCreateContact(client *ChatwootClient, cfg *models.ChatwootConfig, phoneNumber, name, jid string) (*Contact, error) {
	// Primeiro tenta buscar por identifier
	contact, err := client.FindContactByIdentifier(jid)
	if err == nil {
		// Se é grupo e o nome mudou, atualiza no Chatwoot
		if isGroup(jid) && name != "" && contact.Name != name && !strings.HasSuffix(name, "(GROUP)") {
			_ = client.UpdateContact(contact.ID, map[string]interface{}{"name": name})
			contact.Name = name
		}
		return contact, nil
	}

	// Tenta buscar por phone number
	formattedPhone := "+" + phoneNumber
	contact, err = client.FindContactByIdentifier(formattedPhone)
	if err == nil {
		return contact, nil
	}

	// Cria novo contato
	inboxID := cfg.InboxID
	if inboxID == 0 {
		// Tenta buscar inbox
		cwClient := NewChatwootClient(cfg.URL, cfg.Token, cfg.AccountID)
		inboxes, err := cwClient.ListInboxes()
		if err == nil {
			for _, inbox := range inboxes {
				if inbox.Name == cfg.NameInbox {
					inboxID = inbox.ID
					database.DB.Model(cfg).Update("inbox_id", inboxID)
					break
				}
			}
		}
	}

	// Para grupos, não envia phone number (número inválido)
	contactPayload := ContactPayload{
		InboxID:    inboxID,
		Name:       name,
		Identifier: jid,
	}
	if !isGroup(jid) {
		contactPayload.PhoneNumber = formattedPhone
	}

	return client.CreateContact(contactPayload)
}

func findOrCreateConversation(client *ChatwootClient, cfg *models.ChatwootConfig, contact *Contact, remoteJID string) (*models.ChatwootConversation, error) {
	// Busca conversa existente no cache
	var conv models.ChatwootConversation
	if err := database.DB.Where("chatwoot_config_id = ? AND remote_jid = ?", cfg.ID, remoteJID).First(&conv).Error; err == nil {
		// Verifica se precisa reabrir
		if cfg.ReopenConversation {
			_ = client.ToggleConversationStatus(conv.ConversationID, "open")
		}
		return &conv, nil
	}

	// Busca conversas do contato no Chatwoot
	conversations, err := client.GetContactConversations(contact.ID)
	if err == nil {
		for _, c := range conversations {
			if c.InboxID == cfg.InboxID {
				// Encontrou conversa existente
				conv = models.ChatwootConversation{
					ChatwootConfigID: cfg.ID,
					RemoteJID:        remoteJID,
					ContactID:        contact.ID,
					ConversationID:   c.ID,
					InboxID:          c.InboxID,
				}
				database.DB.Create(&conv)

				if cfg.ReopenConversation && c.Status == "resolved" {
					_ = client.ToggleConversationStatus(c.ID, "open")
				}

				return &conv, nil
			}
		}
	}

	// Cria nova conversa
	status := "open"
	if cfg.ConversationPending {
		status = "pending"
	}

	newConv, err := client.CreateConversation(ConversationPayload{
		InboxID:   cfg.InboxID,
		ContactID: contact.ID,
		Status:    status,
		SourceID:  remoteJID,
	})
	if err != nil {
		return nil, err
	}

	conv = models.ChatwootConversation{
		ChatwootConfigID: cfg.ID,
		RemoteJID:        remoteJID,
		ContactID:        contact.ID,
		ConversationID:   newConv.ID,
		InboxID:          newConv.InboxID,
	}
	database.DB.Create(&conv)

	return &conv, nil
}

func extractMessageContent(msgData EvoMessageData) string {
	// Tenta extrair texto baseado no tipo de mensagem
	var msgMap map[string]json.RawMessage
	if err := json.Unmarshal(msgData.Message, &msgMap); err != nil {
		return ""
	}

	// Texto simples
	if text, ok := msgMap["conversation"]; ok {
		var s string
		if err := json.Unmarshal(text, &s); err == nil {
			return s
		}
	}

	// Extended text message
	if ext, ok := msgMap["extendedTextMessage"]; ok {
		var extMsg struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(ext, &extMsg); err == nil {
			return extMsg.Text
		}
	}

	// Image message
	if img, ok := msgMap["imageMessage"]; ok {
		var imgMsg struct {
			Caption string `json:"caption"`
		}
		if err := json.Unmarshal(img, &imgMsg); err == nil {
			return imgMsg.Caption
		}
	}

	// Video message
	if vid, ok := msgMap["videoMessage"]; ok {
		var vidMsg struct {
			Caption string `json:"caption"`
		}
		if err := json.Unmarshal(vid, &vidMsg); err == nil {
			return vidMsg.Caption
		}
	}

	// Document message
	if doc, ok := msgMap["documentMessage"]; ok {
		var docMsg struct {
			Caption  string `json:"caption"`
			FileName string `json:"fileName"`
			Title    string `json:"title"`
		}
		if err := json.Unmarshal(doc, &docMsg); err == nil {
			if docMsg.Caption != "" {
				return docMsg.Caption
			}
			if docMsg.FileName != "" {
				return "📄 " + docMsg.FileName
			}
			return docMsg.Title
		}
	}

	// Audio message
	if _, ok := msgMap["audioMessage"]; ok {
		return "🎵 Áudio"
	}

	// Sticker message
	if _, ok := msgMap["stickerMessage"]; ok {
		return "🏷️ Sticker"
	}

	// Contact message
	if _, ok := msgMap["contactMessage"]; ok {
		return "👤 Contato"
	}

	// Location message
	if loc, ok := msgMap["locationMessage"]; ok {
		var locMsg struct {
			Name    string  `json:"name"`
			Address string  `json:"address"`
			Lat     float64 `json:"degreesLatitude"`
			Lng     float64 `json:"degreesLongitude"`
		}
		if err := json.Unmarshal(loc, &locMsg); err == nil {
			return fmt.Sprintf("📍 %s\n%s\nhttps://maps.google.com/?q=%f,%f", locMsg.Name, locMsg.Address, locMsg.Lat, locMsg.Lng)
		}
	}

	// Reaction
	if reaction, ok := msgMap["reactionMessage"]; ok {
		var r struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(reaction, &r); err == nil {
			return r.Text
		}
	}

	// Poll
	if poll, ok := msgMap["pollCreationMessage"]; ok {
		var p struct {
			Name    string `json:"name"`
			Options []struct {
				OptionName string `json:"optionName"`
			} `json:"options"`
		}
		if err := json.Unmarshal(poll, &p); err == nil {
			text := "📊 " + p.Name + "\n"
			for i, opt := range p.Options {
				text += fmt.Sprintf("%d. %s\n", i+1, opt.OptionName)
			}
			return text
		}
	}

	// Lista/Button response
	if listResp, ok := msgMap["listResponseMessage"]; ok {
		var lr struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(listResp, &lr); err == nil {
			return lr.Title
		}
	}

	// Button response
	if btnResp, ok := msgMap["buttonsResponseMessage"]; ok {
		var br struct {
			SelectedDisplayText string `json:"selectedDisplayText"`
		}
		if err := json.Unmarshal(btnResp, &br); err == nil {
			return br.SelectedDisplayText
		}
	}

	return ""
}

func sendMediaToChatwoot(client *ChatwootClient, conversationID int, caption string, msgData EvoMessageData, messageType string) (*Message, error) {
	if messageType == "" {
		messageType = "incoming"
	}

	if msgData.MediaURL != "" {
		// Baixa a mídia da URL e envia para o Chatwoot
		httpClient := &http.Client{Timeout: 60 * time.Second}
		resp, err := httpClient.Get(msgData.MediaURL)
		if err != nil {
			log.Printf("[IMPA HUB] Erro ao baixar mídia de %s: %v", msgData.MediaURL, err)
			return client.SendMessage(conversationID, MessagePayload{
				Content:     caption + "\n[Mídia não disponível]",
				MessageType: messageType,
			})
		}
		defer resp.Body.Close()

		fileData, err := io.ReadAll(resp.Body)
		if err != nil {
			return client.SendMessage(conversationID, MessagePayload{
				Content:     caption + "\n[Erro ao baixar mídia]",
				MessageType: messageType,
			})
		}

		filename := extractFilename(msgData)
		mimeType := resp.Header.Get("Content-Type")
		if mimeType == "" || mimeType == "application/octet-stream" {
			mimeType = extractMimeType(msgData)
		}

		log.Printf("[IMPA HUB] Enviando mídia para Chatwoot: filename=%s, mimeType=%s, size=%d bytes", filename, mimeType, len(fileData))
		return client.SendMessageWithAttachment(conversationID, caption, messageType, fileData, filename, mimeType)
	}

	// Se tem base64 - decodifica e envia como attachment
	if msgData.Base64 != "" {
		fileData, err := base64.StdEncoding.DecodeString(msgData.Base64)
		if err != nil {
			log.Printf("[IMPA HUB] Erro ao decodificar base64: %v", err)
			return client.SendMessage(conversationID, MessagePayload{
				Content:     caption + "\n[Erro ao processar mídia]",
				MessageType: messageType,
			})
		}

		filename := extractFilename(msgData)
		mimeType := extractMimeType(msgData)

		log.Printf("[IMPA HUB] Enviando mídia base64 para Chatwoot: filename=%s, mimeType=%s, size=%d bytes", filename, mimeType, len(fileData))
		return client.SendMessageWithAttachment(conversationID, caption, messageType, fileData, filename, mimeType)
	}

	// Sem mídia - envia texto
	if caption != "" {
		return client.SendMessage(conversationID, MessagePayload{
			Content:     caption,
			MessageType: messageType,
		})
	}

	return nil, nil
}

// extractMimeType extrai o mimetype de dentro do campo Message
func extractMimeType(msgData EvoMessageData) string {
	// 1. Tenta extrair do campo Message (protobuf)
	if len(msgData.Message) > 0 {
		var msgMap map[string]json.RawMessage
		if err := json.Unmarshal(msgData.Message, &msgMap); err == nil {
			mediaKeys := []string{"imageMessage", "videoMessage", "audioMessage", "documentMessage", "stickerMessage"}
			for _, key := range mediaKeys {
				if raw, ok := msgMap[key]; ok {
					var media struct {
						Mimetype string `json:"mimetype"`
					}
					if json.Unmarshal(raw, &media) == nil && media.Mimetype != "" {
						return media.Mimetype
					}
				}
			}
		}
	}

	// 2. Tenta extrair do messageType (EVO envia formatos como "audio audio/ogg", "image image/jpeg")
	if parts := strings.SplitN(msgData.MessageType, " ", 2); len(parts) == 2 && strings.Contains(parts[1], "/") {
		return parts[1]
	}

	// 3. Fallback baseado no messageType
	mt := strings.ToLower(msgData.MessageType)
	switch {
	case strings.HasPrefix(mt, "image"):
		return "image/jpeg"
	case strings.HasPrefix(mt, "video"):
		return "video/mp4"
	case strings.HasPrefix(mt, "audio"):
		return "audio/ogg; codecs=opus"
	case strings.HasPrefix(mt, "sticker"):
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// min retorna o menor entre dois inteiros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractMimeTypeFromAttachment extrai o MIME type de um attachment Chatwoot
func extractMimeTypeFromAttachment(att ChatwootAttachment) string {
	switch att.FileType {
	case "image":
		return "image/jpeg"
	case "video":
		return "video/mp4"
	case "audio":
		return "audio/mpeg"
	case "file":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// extractAttachmentFilename extrai o nome do arquivo de um attachment Chatwoot
func extractAttachmentFilename(att ChatwootAttachment) string {
	// Tenta extrair da URL data_url (geralmente contém o nome do arquivo)
	if att.DataURL != "" {
		parts := strings.Split(att.DataURL, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			// Remove query parameters se houver
			if idx := strings.Index(filename, "?"); idx != -1 {
				filename = filename[:idx]
			}
			if filename != "" {
				return filename
			}
		}
	}

	// Se não conseguiu extrair da URL, gera um nome baseado no tipo e ID
	extension := ""
	switch att.FileType {
	case "image":
		extension = ".jpg"
	case "video":
		extension = ".mp4"
	case "audio":
		extension = ".mp3"
	case "file":
		extension = ".pdf"
	default:
		extension = ".bin"
	}

	return fmt.Sprintf("attachment_%d%s", att.ID, extension)
}

func extractFilename(msgData EvoMessageData) string {
	var msgMap map[string]json.RawMessage
	if err := json.Unmarshal(msgData.Message, &msgMap); err != nil {
		return "file"
	}

	// Para documentos, usa o nome original se disponível
	if doc, ok := msgMap["documentMessage"]; ok {
		var d struct {
			FileName string `json:"fileName"`
		}
		if err := json.Unmarshal(doc, &d); err == nil && d.FileName != "" {
			return d.FileName
		}
	}

	// Gera nome baseado no MIME type extraído
	mimeType := extractMimeType(msgData)
	ext := mimeToExtension(mimeType)
	random := fmt.Sprintf("%d", time.Now().UnixMilli())

	mt := strings.ToLower(msgData.MessageType)
	switch {
	case strings.HasPrefix(mt, "image"):
		return "image_" + random + ext
	case strings.HasPrefix(mt, "video"):
		return "video_" + random + ext
	case strings.HasPrefix(mt, "audio"):
		return "audio_" + random + ext
	case strings.HasPrefix(mt, "sticker"):
		return "sticker_" + random + ext
	case strings.HasPrefix(mt, "document"):
		return "document_" + random + ext
	default:
		return "file_" + random + ext
	}
}

// mimeToExtension retorna a extensão de arquivo apropriada para o mime type
func mimeToExtension(mimeType string) string {
	// Remove parâmetros (ex: "audio/ogg; codecs=opus" -> "audio/ogg")
	base := strings.Split(mimeType, ";")[0]
	base = strings.TrimSpace(base)

	switch base {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/3gpp":
		return ".3gp"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4":
		return ".m4a"
	case "audio/amr":
		return ".amr"
	case "application/pdf":
		return ".pdf"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "text/plain":
		return ".txt"
	default:
		return ""
	}
}

func cleanJID(jid string) string {
	// Remove @s.whatsapp.net e @g.us
	jid = strings.Split(jid, "@")[0]
	// Remove :XX (device suffix)
	jid = strings.Split(jid, ":")[0]
	return jid
}

// isGroup verifica se o JID é de um grupo do WhatsApp
func isGroup(jid string) bool {
	return strings.HasSuffix(jid, "@g.us")
}

// isIgnoredJID verifica se o JID deve ser ignorado baseado na configuração
func isIgnoredJID(ignoreJSON string, jid string) bool {
	if ignoreJSON == "" {
		return false
	}
	var jids []string
	if json.Unmarshal([]byte(ignoreJSON), &jids) != nil {
		return false
	}
	for _, ignored := range jids {
		if ignored == jid {
			return true
		}
		// Suporte a wildcards: "@g.us" para ignorar todos os grupos
		if strings.HasPrefix(ignored, "@") && strings.HasSuffix(jid, ignored) {
			return true
		}
	}
	return false
}

// isMediaType verifica se a mensagem é de mídia baseado no messageType ou no conteúdo do Message
func isMediaType(msgData EvoMessageData) bool {
	mt := strings.ToLower(msgData.MessageType)
	if strings.HasPrefix(mt, "image") || strings.HasPrefix(mt, "audio") ||
		strings.HasPrefix(mt, "video") || strings.HasPrefix(mt, "document") ||
		strings.HasPrefix(mt, "sticker") || mt == "media" {
		return true
	}
	// Verifica se Message contém chaves de mídia
	var msgMap map[string]json.RawMessage
	if err := json.Unmarshal(msgData.Message, &msgMap); err == nil {
		mediaKeys := []string{"imageMessage", "videoMessage", "audioMessage", "documentMessage", "stickerMessage"}
		for _, key := range mediaKeys {
			if _, ok := msgMap[key]; ok {
				return true
			}
		}
	}
	return false
}

// mediaFallbackText retorna texto de fallback quando a mídia não está disponível
func mediaFallbackText(msgData EvoMessageData) string {
	mt := strings.ToLower(msgData.MessageType)
	switch {
	case strings.HasPrefix(mt, "image"):
		return "📷 Imagem recebida"
	case strings.HasPrefix(mt, "video"):
		return "🎥 Vídeo recebido"
	case strings.HasPrefix(mt, "audio"):
		return "🎵 Áudio recebido"
	case strings.HasPrefix(mt, "document"):
		return "📄 Documento recebido"
	case strings.HasPrefix(mt, "sticker"):
		return "🏷️ Sticker recebido"
	default:
		return "📎 Mídia recebida"
	}
}

func cleanPhoneNumber(phone string) string {
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	return phone
}

func mapFileTypeToMediaType(fileType string) string {
	switch fileType {
	case "image":
		return "image"
	case "audio":
		return "audio"
	case "video":
		return "video"
	default:
		return "document"
	}
}

func logWebhook(instanceID uuid.UUID, event, direction, status, errMsg string) {
	wlog := models.WebhookLog{
		InstanceID: instanceID,
		Event:      event,
		Direction:  direction,
		Status:     status,
		Error:      errMsg,
	}
	database.DB.Create(&wlog)
}
