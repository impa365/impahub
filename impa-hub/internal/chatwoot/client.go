package chatwoot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"time"
)

// ChatwootClient é o client para a API REST do Chatwoot
type ChatwootClient struct {
	BaseURL    string
	Token      string
	AccountID  string
	HTTPClient *http.Client
}

func NewChatwootClient(url, token, accountID string) *ChatwootClient {
	return &ChatwootClient{
		BaseURL:   url,
		Token:     token,
		AccountID: accountID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ==================== Tipos ====================

type Contact struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Identifier  string `json:"identifier,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
}

type ContactPayload struct {
	InboxID     int    `json:"inbox_id"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Identifier  string `json:"identifier,omitempty"`
	Email       string `json:"email,omitempty"`
}

type Conversation struct {
	ID        int    `json:"id"`
	InboxID   int    `json:"inbox_id"`
	Status    string `json:"status"`
	ContactID int    `json:"contact_id"`
	AccountID int    `json:"account_id"`
}

type ConversationPayload struct {
	SourceID  string `json:"source_id,omitempty"`
	InboxID   int    `json:"inbox_id"`
	ContactID int    `json:"contact_id"`
	Status    string `json:"status,omitempty"`
}

type Message struct {
	ID          int             `json:"id"`
	Content     string          `json:"content"`
	MessageType json.RawMessage `json:"message_type"` // pode ser int ou string
	ContentType string          `json:"content_type,omitempty"`
	Private     bool            `json:"private"`
}

type MessagePayload struct {
	Content     string `json:"content"`
	MessageType string `json:"message_type"` // "incoming", "outgoing"
	Private     bool   `json:"private,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

type Inbox struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ChannelID int    `json:"channel_id"`
	Type      string `json:"channel_type"`
}

type InboxPayload struct {
	Name    string       `json:"name"`
	Channel InboxChannel `json:"channel"`
}

type InboxChannel struct {
	Type       string `json:"type"` // "api"
	WebhookURL string `json:"webhook_url,omitempty"`
}

type ConversationList struct {
	Data struct {
		Payload []Conversation `json:"payload"`
	} `json:"data"`
}

type ContactSearchResult struct {
	Payload []Contact `json:"payload"`
}

// ==================== API Methods ====================

func (c *ChatwootClient) apiURL(path string) string {
	return fmt.Sprintf("%s/api/v1/accounts/%s%s", c.BaseURL, c.AccountID, path)
}

func (c *ChatwootClient) doJSON(method, url string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_access_token", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

// ==================== Contacts ====================

func (c *ChatwootClient) CreateContact(payload ContactPayload) (*Contact, error) {
	url := c.apiURL("/contacts")
	data, status, err := c.doJSON("POST", url, payload)
	if err != nil {
		return nil, err
	}

	if status == 422 {
		// Contato já existe - tenta buscar
		return c.FindContactByIdentifier(payload.Identifier)
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao criar contato (status %d): %s", status, string(data))
	}

	var result struct {
		Payload struct {
			Contact Contact `json:"contact"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		// Tenta parsear como contato direto
		var contact Contact
		if err2 := json.Unmarshal(data, &contact); err2 != nil {
			return nil, fmt.Errorf("erro ao parsear contato: %w", err)
		}
		return &contact, nil
	}

	return &result.Payload.Contact, nil
}

func (c *ChatwootClient) FindContactByIdentifier(identifier string) (*Contact, error) {
	url := c.apiURL(fmt.Sprintf("/contacts/search?q=%s&include_contacts=true", identifier))
	data, status, err := c.doJSON("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao buscar contato (status %d)", status)
	}

	var result ContactSearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	for _, contact := range result.Payload {
		if contact.Identifier == identifier || contact.PhoneNumber == identifier {
			return &contact, nil
		}
	}

	return nil, fmt.Errorf("contato não encontrado: %s", identifier)
}

func (c *ChatwootClient) UpdateContact(contactID int, updates map[string]interface{}) error {
	url := c.apiURL(fmt.Sprintf("/contacts/%d", contactID))
	_, status, err := c.doJSON("PUT", url, updates)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("erro ao atualizar contato (status %d)", status)
	}
	return nil
}

// ==================== Conversations ====================

func (c *ChatwootClient) CreateConversation(payload ConversationPayload) (*Conversation, error) {
	url := c.apiURL("/conversations")
	data, status, err := c.doJSON("POST", url, payload)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao criar conversa (status %d): %s", status, string(data))
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, err
	}

	return &conv, nil
}

func (c *ChatwootClient) GetContactConversations(contactID int) ([]Conversation, error) {
	url := c.apiURL(fmt.Sprintf("/contacts/%d/conversations", contactID))
	data, status, err := c.doJSON("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao buscar conversas (status %d)", status)
	}

	var result struct {
		Payload []Conversation `json:"payload"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Payload, nil
}

func (c *ChatwootClient) ToggleConversationStatus(conversationID int, status string) error {
	url := c.apiURL(fmt.Sprintf("/conversations/%d/toggle_status", conversationID))
	_, code, err := c.doJSON("POST", url, map[string]string{"status": status})
	if err != nil {
		return err
	}
	if code >= 400 {
		return fmt.Errorf("erro ao mudar status da conversa (status %d)", code)
	}
	return nil
}

// ==================== Messages ====================

func (c *ChatwootClient) SendMessage(conversationID int, payload MessagePayload) (*Message, error) {
	url := c.apiURL(fmt.Sprintf("/conversations/%d/messages", conversationID))
	data, status, err := c.doJSON("POST", url, payload)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao enviar mensagem (status %d): %s", status, string(data))
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// SendMessageWithAttachment envia uma mensagem com anexo (via multipart form)
func (c *ChatwootClient) SendMessageWithAttachment(conversationID int, content, messageType string, fileData []byte, filename, mimeType string) (*Message, error) {
	url := c.apiURL(fmt.Sprintf("/conversations/%d/messages", conversationID))

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("content", content)
	_ = writer.WriteField("message_type", messageType)

	if fileData != nil && filename != "" {
		// Usa CreatePart com MIME type correto para que Chatwoot renderize inline
		// (CreateFormFile sempre usa application/octet-stream)
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="attachments[]"; filename="%s"`, filename))
		if mimeType != "" {
			h.Set("Content-Type", mimeType)
		} else {
			h.Set("Content-Type", "application/octet-stream")
		}
		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(fileData); err != nil {
			return nil, err
		}
	}

	writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("api_access_token", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("erro ao enviar mensagem com anexo (status %d): %s", resp.StatusCode, string(data))
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (c *ChatwootClient) DeleteMessage(conversationID, messageID int) error {
	url := c.apiURL(fmt.Sprintf("/conversations/%d/messages/%d", conversationID, messageID))
	_, status, err := c.doJSON("DELETE", url, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("erro ao deletar mensagem (status %d)", status)
	}
	return nil
}

// ==================== Inboxes ====================

func (c *ChatwootClient) ListInboxes() ([]Inbox, error) {
	url := c.apiURL("/inboxes")
	data, status, err := c.doJSON("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao listar inboxes (status %d)", status)
	}

	var result struct {
		Payload []Inbox `json:"payload"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Payload, nil
}

func (c *ChatwootClient) CreateInbox(name, webhookURL string) (*Inbox, error) {
	payload := map[string]interface{}{
		"name": name,
		"channel": map[string]interface{}{
			"type":        "api",
			"webhook_url": webhookURL,
		},
	}

	url := c.apiURL("/inboxes")
	data, status, err := c.doJSON("POST", url, payload)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao criar inbox (status %d): %s", status, string(data))
	}

	var inbox Inbox
	if err := json.Unmarshal(data, &inbox); err != nil {
		return nil, err
	}

	return &inbox, nil
}

func (c *ChatwootClient) FindOrCreateInbox(name, webhookURL string) (*Inbox, error) {
	inboxes, err := c.ListInboxes()
	if err != nil {
		return nil, err
	}

	for _, inbox := range inboxes {
		if inbox.Name == name {
			return &inbox, nil
		}
	}

	return c.CreateInbox(name, webhookURL)
}

// ==================== Labels ====================

func (c *ChatwootClient) AddLabelToConversation(conversationID int, labels []string) error {
	url := c.apiURL(fmt.Sprintf("/conversations/%d/labels", conversationID))
	payload := map[string]interface{}{
		"labels": labels,
	}
	_, status, err := c.doJSON("POST", url, payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("erro ao adicionar labels (status %d)", status)
	}
	return nil
}

// ==================== Assignments ====================

func (c *ChatwootClient) AssignConversation(conversationID int, assigneeID int) error {
	url := c.apiURL(fmt.Sprintf("/conversations/%d/assignments", conversationID))
	payload := map[string]interface{}{
		"assignee_id": strconv.Itoa(assigneeID),
	}
	_, status, err := c.doJSON("POST", url, payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("erro ao atribuir conversa (status %d)", status)
	}
	return nil
}
