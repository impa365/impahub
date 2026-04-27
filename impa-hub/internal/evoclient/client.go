package evoclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// Client para comunicação com a API do Evolution GO
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ==================== Tipos de Request/Response ====================

type CreateInstanceRequest struct {
	Name  string `json:"name"`
	Token string `json:"token,omitempty"`
}

type ConnectInstanceRequest struct {
	WebhookURL string   `json:"webhookUrl"`
	Subscribe  []string `json:"subscribe"`
	Immediate  bool     `json:"immediate"`
	Phone      string   `json:"phone,omitempty"`
}

type SendTextRequest struct {
	Number string `json:"number"`
	Text   string `json:"text"`
}

type SendMediaRequest struct {
	Number   string `json:"number"`
	URL      string `json:"url,omitempty"`
	Type     string `json:"type"` // image, audio, video, document
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type EvoResponse struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type InstanceStatusData struct {
	Connected bool   `json:"connected"`
	LoggedIn  bool   `json:"loggedIn"`
	JID       string `json:"jid"`
	Name      string `json:"name"`
}

type QRCodeData struct {
	QRCode string `json:"qrcode"`
	Code   string `json:"code"`
}

type InstanceInfo struct {
	InstanceID   string `json:"id"`
	InstanceName string `json:"name"`
	Token        string `json:"token"`
}

// ==================== Métodos do Client ====================

func (c *Client) doRequest(method, path string, body interface{}, instanceToken string) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao serializar body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Usa token da instância se fornecido, senão usa API key global
	if instanceToken != "" {
		req.Header.Set("apikey", instanceToken)
	} else {
		req.Header.Set("apikey", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("erro na requisição HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// CreateInstance cria uma nova instância no Evolution GO
func (c *Client) CreateInstance(req CreateInstanceRequest) (*InstanceInfo, error) {
	body, status, err := c.doRequest("POST", "/instance/create", req, "")
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao criar instância (status %d): %s", status, string(body))
	}

	var resp struct {
		Message string       `json:"message"`
		Data    InstanceInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta: %w", err)
	}

	return &resp.Data, nil
}

// ConnectInstance conecta uma instância e configura o webhook
func (c *Client) ConnectInstance(instanceToken string, req ConnectInstanceRequest) (json.RawMessage, error) {
	body, status, err := c.doRequest("POST", "/instance/connect", req, instanceToken)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao conectar instância (status %d): %s", status, string(body))
	}

	var resp EvoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta: %w", err)
	}

	return resp.Data, nil
}

// GetInstanceStatus retorna o status da instância
func (c *Client) GetInstanceStatus(instanceToken string) (*InstanceStatusData, error) {
	body, status, err := c.doRequest("GET", "/instance/status", nil, instanceToken)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao obter status (status %d): %s", status, string(body))
	}

	var resp struct {
		Message string             `json:"message"`
		Data    InstanceStatusData `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta: %w", err)
	}

	return &resp.Data, nil
}

// GetQRCode retorna o QR code da instância
func (c *Client) GetQRCode(instanceToken string) (*QRCodeData, error) {
	body, status, err := c.doRequest("GET", "/instance/qr", nil, instanceToken)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao obter QR code (status %d): %s", status, string(body))
	}

	var resp struct {
		Message string     `json:"message"`
		Data    QRCodeData `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta: %w", err)
	}

	return &resp.Data, nil
}

// DisconnectInstance desconecta a instância
func (c *Client) DisconnectInstance(instanceToken string) error {
	_, status, err := c.doRequest("POST", "/instance/disconnect", nil, instanceToken)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("erro ao desconectar (status %d)", status)
	}
	return nil
}

// LogoutInstance faz logout da instância
func (c *Client) LogoutInstance(instanceToken string) error {
	_, status, err := c.doRequest("DELETE", "/instance/logout", nil, instanceToken)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("erro ao fazer logout (status %d)", status)
	}
	return nil
}

// DeleteInstance deleta a instância
func (c *Client) DeleteInstance(instanceID string) error {
	_, status, err := c.doRequest("DELETE", "/instance/delete/"+instanceID, nil, "")
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("erro ao deletar instância (status %d)", status)
	}
	return nil
}

// ListInstances lista todas as instâncias no servidor
func (c *Client) ListInstances() (json.RawMessage, error) {
	body, status, err := c.doRequest("GET", "/instance/all", nil, "")
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("erro ao listar instâncias (status %d): %s", status, string(body))
	}

	var resp EvoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil // Retorna raw se não parsear
	}
	return resp.Data, nil
}

// SendText envia mensagem de texto
func (c *Client) SendText(instanceToken string, req SendTextRequest) (json.RawMessage, error) {
	body, status, err := c.doRequest("POST", "/send/text", req, instanceToken)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("erro ao enviar texto (status %d): %s", status, string(body))
	}
	return body, nil
}

// SendMedia envia mensagem de mídia
func (c *Client) SendMedia(instanceToken string, req SendMediaRequest) (json.RawMessage, error) {
	body, status, err := c.doRequest("POST", "/send/media", req, instanceToken)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("erro ao enviar mídia (status %d): %s", status, string(body))
	}
	return body, nil
}

// SendMediaFile envia mídia via multipart form (upload de arquivo)
func (c *Client) SendMediaFile(instanceToken, number, mediaType, caption, filename string, fileData []byte) (json.RawMessage, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("number", number)
	_ = writer.WriteField("type", mediaType)
	if caption != "" {
		_ = writer.WriteField("caption", caption)
	}
	if filename != "" {
		_ = writer.WriteField("filename", filename)
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, err
	}
	writer.Close()

	url := c.BaseURL + "/send/media"
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("apikey", instanceToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("erro ao enviar arquivo (status %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// ReconnectInstance reconecta a instância
func (c *Client) ReconnectInstance(instanceToken string) error {
	_, status, err := c.doRequest("POST", "/instance/reconnect", nil, instanceToken)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("erro ao reconectar (status %d)", status)
	}
	return nil
}

// GetAdvancedSettings obtém as configurações avançadas da instância
func (c *Client) GetAdvancedSettings(evoInstanceID, instanceToken string) (json.RawMessage, error) {
	path := fmt.Sprintf("/instance/%s/advanced-settings", evoInstanceID)
	body, status, err := c.doRequest("GET", path, nil, instanceToken)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("erro ao obter advanced settings (status %d): %s", status, string(body))
	}
	return body, nil
}

// UpdateAdvancedSettings atualiza as configurações avançadas da instância
func (c *Client) UpdateAdvancedSettings(evoInstanceID, instanceToken string, settings interface{}) (json.RawMessage, error) {
	path := fmt.Sprintf("/instance/%s/advanced-settings", evoInstanceID)
	body, status, err := c.doRequest("PUT", path, settings, instanceToken)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("erro ao atualizar advanced settings (status %d): %s", status, string(body))
	}
	return body, nil
}

// PairInstance solicita código de pareamento via telefone
func (c *Client) PairInstance(instanceToken, phone string) (json.RawMessage, error) {
	reqBody := map[string]interface{}{
		"phone":     phone,
		"subscribe": []string{"ALL"},
	}
	body, status, err := c.doRequest("POST", "/instance/pair", reqBody, instanceToken)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("erro ao solicitar pairing code (status %d): %s", status, string(body))
	}
	return body, nil
}

// SendGeneric envia uma requisição genérica para qualquer endpoint (ex: /send/button, /send/list)
func (c *Client) SendGeneric(instanceToken, path string, body interface{}) (json.RawMessage, error) {
	respBody, status, err := c.doRequest("POST", path, body, instanceToken)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("erro ao enviar %s (status %d): %s", path, status, string(respBody))
	}
	return respBody, nil
}

// GroupInfoResponse representa a resposta da API de info do grupo
// A API retorna: {"data": {"Name": "...", "Topic": "...", ...}, "message": "success"}
type GroupInfoResponse struct {
	JID       string `json:"JID"`
	Name      string `json:"Name"`
	NameSetAt string `json:"NameSetAt"`
	Topic     string `json:"Topic"`
}

// GetGroupInfo obtém informações de um grupo (nome, tópico, etc.)
func (c *Client) GetGroupInfo(instanceToken string, groupJID string) (*GroupInfoResponse, error) {
	reqBody := map[string]string{"groupJid": groupJID}
	body, status, err := c.doRequest("POST", "/group/info", reqBody, instanceToken)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("erro ao obter info do grupo (status %d): %s", status, string(body))
	}

	var resp struct {
		Message string            `json:"message"`
		Data    GroupInfoResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("erro ao parsear info do grupo: %w", err)
	}

	return &resp.Data, nil
}
