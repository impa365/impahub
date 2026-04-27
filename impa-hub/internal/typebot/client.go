package typebot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TypebotClient faz chamadas HTTP à API do Typebot
type TypebotClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewTypebotClient(baseURL string) *TypebotClient {
	return &TypebotClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ==================== Tipos de Request/Response ====================

type StartChatRequest struct {
	PrefilledVariables map[string]interface{} `json:"prefilledVariables,omitempty"`
}

type ContinueChatRequest struct {
	Message string `json:"message"`
}

type TypebotResponse struct {
	SessionID         string             `json:"sessionId"`
	Messages          []TypebotMessage   `json:"messages"`
	Input             *TypebotInput      `json:"input,omitempty"`
	ClientSideActions []ClientSideAction `json:"clientSideActions,omitempty"`
}

type TypebotMessage struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"` // text, image, video, audio
	Content json.RawMessage `json:"content"`
}

type TypebotTextContent struct {
	RichText []RichTextBlock `json:"richText"`
}

type RichTextBlock struct {
	Children []RichTextElement `json:"children"`
}

type RichTextElement struct {
	Text      string            `json:"text,omitempty"`
	Bold      bool              `json:"bold,omitempty"`
	Italic    bool              `json:"italic,omitempty"`
	Underline bool              `json:"underline,omitempty"`
	URL       string            `json:"url,omitempty"`
	Type      string            `json:"type,omitempty"` // p, a, ol, li, inline-variable
	Children  []RichTextElement `json:"children,omitempty"`
}

type TypebotMediaContent struct {
	URL string `json:"url"`
}

type TypebotInput struct {
	Type  string          `json:"type"` // choice input, text input
	ID    string          `json:"id"`
	Items []TypebotChoice `json:"items,omitempty"`
}

type TypebotChoice struct {
	Content string `json:"content"`
}

type ClientSideAction struct {
	LastBubbleBlockID string      `json:"lastBubbleBlockId,omitempty"`
	Wait              *WaitAction `json:"wait,omitempty"`
}

type WaitAction struct {
	SecondsToWaitFor int `json:"secondsToWaitFor"`
}

// ==================== Métodos da API ====================

func (c *TypebotClient) doRequest(method, path string, body interface{}) ([]byte, int, error) {
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

// StartChat inicia uma nova sessão com o Typebot
// POST /api/v1/typebots/{typebotId}/startChat
func (c *TypebotClient) StartChat(typebotID string, variables map[string]interface{}) (*TypebotResponse, error) {
	path := fmt.Sprintf("/api/v1/typebots/%s/startChat", typebotID)

	reqBody := StartChatRequest{
		PrefilledVariables: variables,
	}

	body, status, err := c.doRequest("POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao iniciar chat (status %d): %s", status, string(body))
	}

	var resp TypebotResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta: %w", err)
	}

	return &resp, nil
}

// ContinueChat continua uma sessão existente
// POST /api/v1/sessions/{sessionId}/continueChat
func (c *TypebotClient) ContinueChat(sessionID, message string) (*TypebotResponse, error) {
	path := fmt.Sprintf("/api/v1/sessions/%s/continueChat", sessionID)

	reqBody := ContinueChatRequest{
		Message: message,
	}

	body, status, err := c.doRequest("POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, fmt.Errorf("erro ao continuar chat (status %d): %s", status, string(body))
	}

	var resp TypebotResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta: %w", err)
	}

	return &resp, nil
}
