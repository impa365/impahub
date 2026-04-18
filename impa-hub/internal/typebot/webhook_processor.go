package typebot

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/evoclient"
	"github.com/impa-hub/internal/models"
)

// ==================== Debounce ====================

type debounceEntry struct {
	message string
	timer   *time.Timer
}

var (
	debounceMap = make(map[string]*debounceEntry)
	debounceMu  sync.Mutex
)

// ==================== Webhook Payload Types ====================

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
}

type EvoMessageInfo struct {
	ID        string `json:"ID"`
	Chat      string `json:"Chat"`
	Sender    string `json:"Sender"`
	Timestamp string `json:"Timestamp"`
	IsFromMe  bool   `json:"IsFromMe"`
	PushName  string `json:"PushName,omitempty"`
}

// ==================== Main Entry Point ====================

// ProcessEvoWebhook é chamado pelo webhook handler quando chega um evento do Evolution GO
func ProcessEvoWebhook(instanceID uuid.UUID, payload EvoWebhookPayload) {
	// Só processa eventos de mensagem
	if payload.Event != "Message" && payload.Event != "SendMessage" {
		return
	}

	// Busca instância
	var inst models.Instance
	if err := database.DB.Where("id = ?", instanceID).First(&inst).Error; err != nil {
		return
	}

	// Busca TODOS os configs Typebot ativos para esta instância
	var configs []models.TypebotConfig
	if err := database.DB.Where("instance_id = ? AND enabled = ?", instanceID, true).Find(&configs).Error; err != nil || len(configs) == 0 {
		return // Sem config Typebot ativa
	}

	// Busca settings globais
	var settings models.TypebotSetting
	hasSettings := database.DB.Where("instance_id = ?", instanceID).First(&settings).Error == nil

	// Parse dos dados da mensagem
	var msgData EvoMessageData
	if err := json.Unmarshal(payload.Data, &msgData); err != nil {
		log.Printf("[TYPEBOT] Erro ao parsear dados da mensagem: %v", err)
		return
	}

	// Ignora broadcast
	if msgData.Info.Chat == "status@broadcast" || strings.HasSuffix(msgData.Info.Chat, "@broadcast") {
		return
	}

	// Ignora mensagens de grupo por padrão
	if isGroup(msgData.Info.Chat) {
		log.Printf("[TYPEBOT] Ignorando mensagem de grupo: %s", msgData.Info.Chat)
		return
	}

	remoteJID := msgData.Info.Chat

	// Verifica ignore JIDs (global settings)
	if hasSettings && isIgnoredJID(settings.IgnoreJids, remoteJID) {
		return
	}

	// Verifica se é mensagem enviada por mim
	if msgData.Info.IsFromMe {
		// Verifica global setting primeiro
		listeningFromMe := false
		stopBotFromMe := false

		// Busca sessão ativa para verificar config do bot
		session, cfg := findActiveSession(configs, remoteJID)
		if session != nil && cfg != nil {
			listeningFromMe = cfg.ListeningFromMe
			stopBotFromMe = cfg.StopBotFromMe
		} else if hasSettings {
			listeningFromMe = settings.ListeningFromMe
			stopBotFromMe = settings.StopBotFromMe
		}

		if !listeningFromMe {
			if stopBotFromMe && session != nil {
				pauseSession(session.TypebotConfigID, remoteJID)
			}
			return
		}
	}

	// Extrai conteúdo da mensagem
	content := extractMessageContent(msgData)
	if content == "" {
		return
	}

	pushName := msgData.Info.PushName

	// Busca sessão ativa
	session, activeCfg := findActiveSession(configs, remoteJID)

	// Determina config efetiva (merge settings → bot config)
	var effectiveCfg models.TypebotConfig
	if activeCfg != nil {
		effectiveCfg = mergeSettings(*activeCfg, settings, hasSettings)
	}

	// Debounce
	debounceTime := 0
	if session != nil && activeCfg != nil {
		debounceTime = effectiveCfg.DebounceTime
	} else {
		// Para nova sessão, usa settings globais
		if hasSettings {
			debounceTime = settings.DebounceTime
		}
	}

	if debounceTime > 0 {
		processDebounce(remoteJID, content, debounceTime, func(groupedMsg string) {
			processMessageMultiBot(inst, configs, settings, hasSettings, remoteJID, pushName, groupedMsg, payload.InstanceToken)
		})
		return
	}

	processMessageMultiBot(inst, configs, settings, hasSettings, remoteJID, pushName, content, payload.InstanceToken)
}

// ==================== Multi-Bot Message Processing ====================

// findActiveSession busca sessão aberta entre todas as configs
func findActiveSession(configs []models.TypebotConfig, remoteJID string) (*models.TypebotSession, *models.TypebotConfig) {
	for _, cfg := range configs {
		var session models.TypebotSession
		if database.DB.Where("typebot_config_id = ? AND remote_jid = ? AND status = ?",
			cfg.ID, remoteJID, models.TypebotSessionOpened).First(&session).Error == nil {
			return &session, &cfg
		}
	}
	return nil, nil
}

// findBotByTrigger busca o bot correto baseado no conteúdo da mensagem
// Prioridade: 1) trigger "all" → 2) keyword/advanced match → 3) fallback
func findBotByTrigger(configs []models.TypebotConfig, content string, settings models.TypebotSetting, hasSettings bool) *models.TypebotConfig {
	var allTriggerBot *models.TypebotConfig

	for i := range configs {
		cfg := &configs[i]
		switch cfg.TriggerType {
		case "all":
			allTriggerBot = cfg
		case "keyword", "advanced":
			if matchesTrigger(*cfg, content) {
				return cfg
			}
		}
	}

	// Se tem "all", usa
	if allTriggerBot != nil {
		return allTriggerBot
	}

	// Tenta fallback dos settings
	if hasSettings && settings.TypebotIDFallback != nil {
		for i := range configs {
			if configs[i].ID == *settings.TypebotIDFallback {
				return &configs[i]
			}
		}
		// Fallback pode ser um bot desabilitado, busca direto do DB
		var fallback models.TypebotConfig
		if database.DB.First(&fallback, settings.TypebotIDFallback).Error == nil {
			return &fallback
		}
	}

	return nil
}

// mergeSettings mescla settings globais com config do bot (bot tem prioridade)
func mergeSettings(cfg models.TypebotConfig, settings models.TypebotSetting, hasSettings bool) models.TypebotConfig {
	if !hasSettings {
		return cfg
	}
	// Se o bot não define valor próprio, herda do settings
	if cfg.Expire == 0 && settings.Expire > 0 {
		cfg.Expire = settings.Expire
	}
	if cfg.KeywordFinish == "" && settings.KeywordFinish != "" {
		cfg.KeywordFinish = settings.KeywordFinish
	}
	if cfg.DelayMessage == 0 && settings.DelayMessage > 0 {
		cfg.DelayMessage = settings.DelayMessage
	}
	if cfg.UnknownMessage == "" && settings.UnknownMessage != "" {
		cfg.UnknownMessage = settings.UnknownMessage
	}
	if cfg.DebounceTime == 0 && settings.DebounceTime > 0 {
		cfg.DebounceTime = settings.DebounceTime
	}
	return cfg
}

// isGroup verifica se o JID é de um grupo do WhatsApp
func isGroup(jid string) bool {
	return strings.HasSuffix(jid, "@g.us")
}

// isIgnoredJID verifica se um JID está na lista de ignorados
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

func processMessageMultiBot(inst models.Instance, configs []models.TypebotConfig, settings models.TypebotSetting, hasSettings bool, remoteJID, pushName, content, instanceToken string) {
	// 1. Busca sessão ativa
	session, activeCfg := findActiveSession(configs, remoteJID)

	if session != nil && activeCfg != nil {
		// Sessão existe — verifica ignore JIDs do bot
		if isIgnoredJID(activeCfg.IgnoreJids, remoteJID) {
			return
		}

		effectiveCfg := mergeSettings(*activeCfg, settings, hasSettings)

		// Verifica keyword de finalização
		if effectiveCfg.KeywordFinish != "" && strings.EqualFold(content, effectiveCfg.KeywordFinish) {
			closeSession(session, effectiveCfg.KeepOpen)
			return
		}

		// Verifica expiração
		if effectiveCfg.Expire > 0 {
			expireTime := session.UpdatedAt.Add(time.Duration(effectiveCfg.Expire) * time.Minute)
			if time.Now().After(expireTime) {
				closeSession(session, effectiveCfg.KeepOpen)
				session = nil
				activeCfg = nil
				// Cai no fluxo de nova sessão abaixo
			}
		}

		if session != nil {
			// Continua sessão existente
			continueExistingSession(inst, session, effectiveCfg, configs, settings, hasSettings, remoteJID, pushName, content, instanceToken)
			return
		}
	}

	// 2. Sem sessão ativa — busca bot por trigger
	bot := findBotByTrigger(configs, content, settings, hasSettings)
	if bot == nil {
		return
	}

	// Verifica ignore JIDs do bot
	if isIgnoredJID(bot.IgnoreJids, remoteJID) {
		return
	}

	effectiveCfg := mergeSettings(*bot, settings, hasSettings)

	// Busca EvoServer
	var evoServer models.EvoServer
	if err := database.DB.First(&evoServer, inst.EvoServerID).Error; err != nil {
		log.Printf("[TYPEBOT] Erro ao buscar EvoServer: %v", err)
		return
	}
	evoClient := evoclient.New(evoServer.URL, evoServer.APIKey)

	// Cria nova sessão
	newSession, typebotResp, err := createNewSession(inst, effectiveCfg, remoteJID, pushName)
	if err != nil {
		log.Printf("[TYPEBOT] Erro ao criar sessão: %v", err)
		return
	}

	sendTypebotResponses(evoClient, instanceToken, remoteJID, newSession, effectiveCfg, typebotResp)
}

func continueExistingSession(inst models.Instance, session *models.TypebotSession, cfg models.TypebotConfig, configs []models.TypebotConfig, settings models.TypebotSetting, hasSettings bool, remoteJID, pushName, content, instanceToken string) {
	var evoServer models.EvoServer
	if err := database.DB.First(&evoServer, inst.EvoServerID).Error; err != nil {
		log.Printf("[TYPEBOT] Erro ao buscar EvoServer: %v", err)
		return
	}
	evoClient := evoclient.New(evoServer.URL, evoServer.APIKey)

	typebotClient := NewTypebotClient(cfg.URL)

	// Extrai sessionId real
	parts := strings.SplitN(session.SessionID, "-", 2)
	if len(parts) < 2 {
		log.Printf("[TYPEBOT] SessionID inválido: %s", session.SessionID)
		return
	}
	typebotSessionID := parts[1]

	database.DB.Model(session).Update("await_user", false)

	resp, err := typebotClient.ContinueChat(typebotSessionID, content)
	if err != nil {
		log.Printf("[TYPEBOT] Erro ao continuar chat: %v", err)
		// Tenta recriar sessão
		closeSession(session, false)
		bot := findBotByTrigger(configs, content, settings, hasSettings)
		if bot != nil {
			effectiveCfg := mergeSettings(*bot, settings, hasSettings)
			newSession, typebotResp, err := createNewSession(inst, effectiveCfg, remoteJID, pushName)
			if err != nil {
				log.Printf("[TYPEBOT] Erro ao recriar sessão: %v", err)
				return
			}
			sendTypebotResponses(evoClient, instanceToken, remoteJID, newSession, effectiveCfg, typebotResp)
		}
		return
	}

	sendTypebotResponses(evoClient, instanceToken, remoteJID, session, cfg, resp)
}

// ==================== Session Management ====================

func createNewSession(inst models.Instance, cfg models.TypebotConfig, remoteJID, pushName string) (*models.TypebotSession, *TypebotResponse, error) {
	typebotClient := NewTypebotClient(cfg.URL)

	// Prepara variáveis pré-preenchidas
	variables := map[string]interface{}{
		"remoteJid":    remoteJID,
		"pushName":     pushName,
		"instanceName": inst.EvoInstanceName,
	}

	// Adiciona variáveis customizadas da config
	if cfg.PrefilledVariables != "" {
		var custom map[string]interface{}
		if err := json.Unmarshal([]byte(cfg.PrefilledVariables), &custom); err == nil {
			for k, v := range custom {
				variables[k] = v
			}
		}
	}

	resp, err := typebotClient.StartChat(cfg.Typebot, variables)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao iniciar chat no Typebot: %w", err)
	}

	if resp.SessionID == "" {
		return nil, nil, fmt.Errorf("Typebot não retornou sessionId")
	}

	// Gera ID local + sessionId do Typebot
	localID := fmt.Sprintf("%d", rand.Intn(10000000000))
	compositeSessionID := fmt.Sprintf("%s-%s", localID, resp.SessionID)

	// Serializa parâmetros
	paramsJSON, _ := json.Marshal(variables)

	session := models.TypebotSession{
		TypebotConfigID: cfg.ID,
		RemoteJID:       remoteJID,
		PushName:        pushName,
		SessionID:       compositeSessionID,
		Status:          models.TypebotSessionOpened,
		AwaitUser:       false,
		Parameters:      string(paramsJSON),
	}

	if err := database.DB.Create(&session).Error; err != nil {
		return nil, nil, fmt.Errorf("erro ao salvar sessão: %w", err)
	}

	log.Printf("[TYPEBOT] Nova sessão criada: remoteJID=%s sessionID=%s", remoteJID, compositeSessionID)

	return &session, resp, nil
}

func closeSession(session *models.TypebotSession, keepOpen bool) {
	if keepOpen {
		database.DB.Model(session).Update("status", models.TypebotSessionClosed)
		log.Printf("[TYPEBOT] Sessão fechada (keepOpen): %s", session.RemoteJID)
	} else {
		database.DB.Delete(session)
		log.Printf("[TYPEBOT] Sessão deletada: %s", session.RemoteJID)
	}
}

func pauseSession(configID uuid.UUID, remoteJID string) {
	database.DB.Model(&models.TypebotSession{}).
		Where("typebot_config_id = ? AND remote_jid = ? AND status = ?",
			configID, remoteJID, models.TypebotSessionOpened).
		Update("status", models.TypebotSessionPaused)
	log.Printf("[TYPEBOT] Sessão pausada: %s", remoteJID)
}

// ==================== Trigger Matching ====================

func matchesTrigger(cfg models.TypebotConfig, content string) bool {
	switch cfg.TriggerType {
	case "all":
		return true
	case "none":
		return false
	case "advanced":
		if cfg.TriggerValue == "" {
			return false
		}
		matched, _ := regexp.MatchString(cfg.TriggerValue, content)
		return matched
	case "keyword":
		if cfg.TriggerValue == "" {
			return false
		}
		contentLower := strings.ToLower(content)
		triggerLower := strings.ToLower(cfg.TriggerValue)

		switch cfg.TriggerOperator {
		case "equals":
			return contentLower == triggerLower
		case "startsWith":
			return strings.HasPrefix(contentLower, triggerLower)
		case "endsWith":
			return strings.HasSuffix(contentLower, triggerLower)
		case "regex":
			matched, _ := regexp.MatchString(cfg.TriggerValue, content)
			return matched
		default: // contains
			return strings.Contains(contentLower, triggerLower)
		}
	default:
		return false
	}
}

// ==================== Send Typebot Responses to WhatsApp ====================

func sendTypebotResponses(evoClient *evoclient.Client, instanceToken, remoteJID string, session *models.TypebotSession, cfg models.TypebotConfig, resp *TypebotResponse) {
	if resp == nil {
		return
	}

	// Processa cada mensagem do Typebot
	for _, msg := range resp.Messages {
		switch msg.Type {
		case "text":
			processTextMessage(evoClient, instanceToken, remoteJID, msg, cfg)
		case "image":
			processMediaMessage(evoClient, instanceToken, remoteJID, msg, "image", cfg)
		case "video":
			processMediaMessage(evoClient, instanceToken, remoteJID, msg, "video", cfg)
		case "audio":
			processAudioMessage(evoClient, instanceToken, remoteJID, msg, cfg)
		}

		// Aplica delay do clientSideActions (wait)
		waitSeconds := findWaitSeconds(resp.ClientSideActions, msg.ID)
		if waitSeconds > 0 {
			time.Sleep(time.Duration(waitSeconds) * time.Second)
		}
	}

	// Processa input (choice input)
	if resp.Input != nil && resp.Input.Type == "choice input" && len(resp.Input.Items) > 0 {
		var choiceText string
		for _, item := range resp.Input.Items {
			choiceText += fmt.Sprintf("▶️ %s\n", item.Content)
		}
		choiceText = strings.TrimRight(choiceText, "\n")

		sendTextOrInteractive(evoClient, instanceToken, remoteJID, choiceText, cfg)

		// Marca sessão aguardando resposta
		database.DB.Model(session).Update("await_user", true)
	} else if resp.Input != nil {
		// Input de texto (aberto) - aguarda resposta
		database.DB.Model(session).Update("await_user", true)
	} else {
		// Sem input = conversa encerrada pelo Typebot
		closeSession(session, cfg.KeepOpen)
	}
}

func processTextMessage(evoClient *evoclient.Client, instanceToken, remoteJID string, msg TypebotMessage, cfg models.TypebotConfig) {
	var textContent TypebotTextContent
	if err := json.Unmarshal(msg.Content, &textContent); err != nil {
		log.Printf("[TYPEBOT] Erro ao parsear texto: %v", err)
		return
	}

	formattedText := formatRichText(textContent.RichText)

	// Detecta tipos especiais
	if strings.Contains(formattedText, "[buttons]") {
		sendButtonMessage(evoClient, instanceToken, remoteJID, formattedText)
		return
	}
	if strings.Contains(formattedText, "[list]") {
		sendListMessage(evoClient, instanceToken, remoteJID, formattedText)
		return
	}

	sendTextOrInteractive(evoClient, instanceToken, remoteJID, formattedText, cfg)
}

func processMediaMessage(evoClient *evoclient.Client, instanceToken, remoteJID string, msg TypebotMessage, mediaType string, cfg models.TypebotConfig) {
	var mediaContent TypebotMediaContent
	if err := json.Unmarshal(msg.Content, &mediaContent); err != nil {
		log.Printf("[TYPEBOT] Erro ao parsear mídia: %v", err)
		return
	}

	if cfg.DelayMessage > 0 {
		time.Sleep(time.Duration(cfg.DelayMessage) * time.Millisecond)
	}

	evoClient.SendMedia(instanceToken, evoclient.SendMediaRequest{
		Number: remoteJID,
		URL:    mediaContent.URL,
		Type:   mediaType,
	})
}

func processAudioMessage(evoClient *evoclient.Client, instanceToken, remoteJID string, msg TypebotMessage, cfg models.TypebotConfig) {
	var mediaContent TypebotMediaContent
	if err := json.Unmarshal(msg.Content, &mediaContent); err != nil {
		log.Printf("[TYPEBOT] Erro ao parsear áudio: %v", err)
		return
	}

	if cfg.DelayMessage > 0 {
		time.Sleep(time.Duration(cfg.DelayMessage) * time.Millisecond)
	}

	evoClient.SendMedia(instanceToken, evoclient.SendMediaRequest{
		Number: remoteJID,
		URL:    mediaContent.URL,
		Type:   "audio",
	})
}

// ==================== Text Formatting ====================

func formatRichText(blocks []RichTextBlock) string {
	var result string
	for _, block := range blocks {
		for _, element := range block.Children {
			result += applyFormatting(element)
		}
		result += "\n"
	}

	// Limpar formatação duplicada
	result = strings.ReplaceAll(result, "**", "")
	result = strings.ReplaceAll(result, "__", "")
	result = strings.ReplaceAll(result, "~~", "")
	result = strings.TrimRight(result, "\n")

	return result
}

func applyFormatting(element RichTextElement) string {
	var text string

	if element.Text != "" {
		text += element.Text
	}

	// Processa filhos recursivamente (exceto links)
	if len(element.Children) > 0 && element.Type != "a" {
		for _, child := range element.Children {
			text += applyFormatting(child)
		}
	}

	// Tratamento por tipo
	if element.Type == "p" {
		text = strings.TrimSpace(text) + "\n"
	}

	if element.Type == "inline-variable" {
		text = strings.TrimSpace(text)
	}

	if element.Type == "ol" {
		lines := strings.Split(text, "\n")
		var numbered string
		idx := 1
		for _, line := range lines {
			if line != "" {
				numbered += fmt.Sprintf("%d. %s\n", idx, line)
				idx++
			}
		}
		text = "\n" + numbered
	}

	if element.Type == "li" {
		lines := strings.Split(text, "\n")
		var indented string
		for _, line := range lines {
			if line != "" {
				indented += fmt.Sprintf("  %s\n", line)
			}
		}
		text = indented
	}

	// Aplica formatação WhatsApp
	var formats string
	if element.Bold {
		formats += "*"
	}
	if element.Italic {
		formats += "_"
	}
	if element.Underline {
		formats += "~"
	}

	if formats != "" {
		reversed := reverseString(formats)
		text = formats + text + reversed
	}

	// Links
	if element.URL != "" {
		if len(element.Children) > 0 && element.Children[0].Text != "" {
			text = fmt.Sprintf("[%s]\n(%s)", text, element.URL)
		} else {
			text = element.URL
		}
	}

	return text
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ==================== Interactive Messages ====================

func sendTextOrInteractive(evoClient *evoclient.Client, instanceToken, remoteJID, text string, cfg models.TypebotConfig) {
	if cfg.DelayMessage > 0 {
		time.Sleep(time.Duration(cfg.DelayMessage) * time.Millisecond)
	}

	evoClient.SendText(instanceToken, evoclient.SendTextRequest{
		Number: remoteJID,
		Text:   text,
	})
}

func sendButtonMessage(evoClient *evoclient.Client, instanceToken, remoteJID, formattedText string) {
	buttonJSON := map[string]interface{}{
		"number": remoteJID,
	}

	// Parse campos
	if m := regexp.MustCompile(`\[title\]([\s\S]*?)(?:\[)`).FindStringSubmatch(formattedText); len(m) > 1 {
		buttonJSON["title"] = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`\[description\]([\s\S]*?)(?:\[)`).FindStringSubmatch(formattedText); len(m) > 1 {
		buttonJSON["description"] = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`\[footer\]([\s\S]*?)(?:\[)`).FindStringSubmatch(formattedText); len(m) > 1 {
		buttonJSON["footer"] = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`\[thumbnailUrl\]([\s\S]*?)(?:\[)`).FindStringSubmatch(formattedText); len(m) > 1 {
		buttonJSON["imageUrl"] = strings.TrimSpace(m[1])
	}

	// Parse botões
	var buttons []map[string]interface{}

	buttonTypes := map[string]*regexp.Regexp{
		"reply": regexp.MustCompile(`\[reply\]([\s\S]*?)(?:\[(?:reply|pix|copy|call|url)|$)`),
		"pix":   regexp.MustCompile(`\[pix\]([\s\S]*?)(?:\[(?:reply|pix|copy|call|url)|$)`),
		"copy":  regexp.MustCompile(`\[copy\]([\s\S]*?)(?:\[(?:reply|pix|copy|call|url)|$)`),
		"call":  regexp.MustCompile(`\[call\]([\s\S]*?)(?:\[(?:reply|pix|copy|call|url)|$)`),
		"url":   regexp.MustCompile(`\[url\]([\s\S]*?)(?:\[(?:reply|pix|copy|call|url)|$)`),
	}

	for btnType, pattern := range buttonTypes {
		matches := pattern.FindAllStringSubmatch(formattedText, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			btnContent := strings.TrimSpace(match[1])
			btn := map[string]interface{}{"type": btnType}

			switch btnType {
			case "reply":
				btn["displayText"] = extractField(btnContent, "displayText")
				btn["id"] = extractField(btnContent, "id")
			case "pix":
				btn["currency"] = extractField(btnContent, "currency")
				btn["name"] = extractField(btnContent, "name")
				btn["keyType"] = extractField(btnContent, "keyType")
				btn["key"] = extractField(btnContent, "key")
			case "copy":
				btn["displayText"] = extractField(btnContent, "displayText")
				btn["copyCode"] = extractField(btnContent, "copyCode")
			case "call":
				btn["displayText"] = extractField(btnContent, "displayText")
				btn["phoneNumber"] = extractField(btnContent, "phone")
			case "url":
				btn["displayText"] = extractField(btnContent, "displayText")
				btn["url"] = extractField(btnContent, "url")
			}

			buttons = append(buttons, btn)
		}
	}

	buttonJSON["buttons"] = buttons

	// Envia via Evolution GO /send/button
	evoClient.SendGeneric(instanceToken, "/send/button", buttonJSON)
}

func sendListMessage(evoClient *evoclient.Client, instanceToken, remoteJID, formattedText string) {
	listJSON := map[string]interface{}{
		"number": remoteJID,
	}

	if m := regexp.MustCompile(`\[title\]([\s\S]*?)(?:\[description\])`).FindStringSubmatch(formattedText); len(m) > 1 {
		listJSON["title"] = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`\[description\]([\s\S]*?)(?:\[buttonText\])`).FindStringSubmatch(formattedText); len(m) > 1 {
		listJSON["description"] = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`\[buttonText\]([\s\S]*?)(?:\[footerText\])`).FindStringSubmatch(formattedText); len(m) > 1 {
		listJSON["buttonText"] = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`\[footerText\]([\s\S]*?)(?:\[menu\])`).FindStringSubmatch(formattedText); len(m) > 1 {
		listJSON["footerText"] = strings.TrimSpace(m[1])
	}

	// Parse sections
	menuMatch := regexp.MustCompile(`\[menu\]([\s\S]*?)\[/menu\]`).FindStringSubmatch(formattedText)
	if len(menuMatch) > 1 {
		sectionPattern := regexp.MustCompile(`\[section\]([\s\S]*?)(?:\[section\]|\[/section\]|\[/menu\])`)
		sectionMatches := sectionPattern.FindAllStringSubmatch(menuMatch[1], -1)

		var sections []map[string]interface{}
		for _, sm := range sectionMatches {
			if len(sm) < 2 {
				continue
			}
			sectionContent := sm[1]
			sectionTitle := extractField(sectionContent, "title")

			rowPattern := regexp.MustCompile(`\[row\]([\s\S]*?)(?:\[row\]|\[/row\]|\[/section\]|\[/menu\])`)
			rowMatches := rowPattern.FindAllStringSubmatch(sectionContent, -1)

			var rows []map[string]interface{}
			for _, rm := range rowMatches {
				if len(rm) < 2 {
					continue
				}
				rowContent := rm[1]
				rows = append(rows, map[string]interface{}{
					"title":       extractField(rowContent, "title"),
					"description": extractField(rowContent, "description"),
					"rowId":       extractField(rowContent, "rowId"),
				})
			}

			sections = append(sections, map[string]interface{}{
				"title": sectionTitle,
				"rows":  rows,
			})
		}
		listJSON["sections"] = sections
	}

	evoClient.SendGeneric(instanceToken, "/send/list", listJSON)
}

// ==================== Message Content Extraction ====================

func extractMessageContent(msgData EvoMessageData) string {
	// Parse da mensagem para extrair texto
	var msgMap map[string]interface{}
	if err := json.Unmarshal(msgData.Message, &msgMap); err != nil {
		return ""
	}

	// Texto simples (conversation)
	if conv, ok := msgMap["conversation"].(string); ok && conv != "" {
		return conv
	}

	// ExtendedTextMessage
	if ext, ok := msgMap["extendedTextMessage"].(map[string]interface{}); ok {
		if text, ok := ext["text"].(string); ok {
			return text
		}
	}

	// Resposta de botão (buttonsResponseMessage)
	if btn, ok := msgMap["buttonsResponseMessage"].(map[string]interface{}); ok {
		if text, ok := btn["selectedDisplayText"].(string); ok {
			return text
		}
		if text, ok := btn["selectedButtonId"].(string); ok {
			return text
		}
	}

	// Resposta de lista (listResponseMessage)
	if list, ok := msgMap["listResponseMessage"].(map[string]interface{}); ok {
		if sr, ok := list["singleSelectReply"].(map[string]interface{}); ok {
			if rowID, ok := sr["selectedRowId"].(string); ok {
				return rowID
			}
		}
		if title, ok := list["title"].(string); ok {
			return title
		}
	}

	// InteractiveResponseMessage (nativeFlowResponse ou botão interativo)
	if interactive, ok := msgMap["interactiveResponseMessage"].(map[string]interface{}); ok {
		if body, ok := interactive["body"].(map[string]interface{}); ok {
			if text, ok := body["text"].(string); ok {
				return text
			}
		}
		// nativeFlowResponseMessage
		if nfr, ok := interactive["nativeFlowResponseMessage"].(map[string]interface{}); ok {
			if paramsJSON, ok := nfr["paramsJson"].(string); ok {
				var params map[string]interface{}
				if json.Unmarshal([]byte(paramsJSON), &params) == nil {
					if id, ok := params["id"].(string); ok {
						return id
					}
				}
				return paramsJSON
			}
		}
	}

	// Mensagem de imagem com caption
	if img, ok := msgMap["imageMessage"].(map[string]interface{}); ok {
		if caption, ok := img["caption"].(string); ok && caption != "" {
			return caption
		}
	}

	// Mensagem de vídeo com caption
	if vid, ok := msgMap["videoMessage"].(map[string]interface{}); ok {
		if caption, ok := vid["caption"].(string); ok && caption != "" {
			return caption
		}
	}

	// Mensagem de documento com caption
	if doc, ok := msgMap["documentMessage"].(map[string]interface{}); ok {
		if caption, ok := doc["caption"].(string); ok && caption != "" {
			return caption
		}
	}

	return ""
}

// ==================== Debounce ====================

func processDebounce(remoteJID, content string, debounceSeconds int, callback func(string)) {
	debounceMu.Lock()
	defer debounceMu.Unlock()

	if entry, exists := debounceMap[remoteJID]; exists {
		// Já tem mensagem pendente - concatena
		entry.message += "\n" + content
		entry.timer.Stop()
		log.Printf("[TYPEBOT] Debounce: mensagem agrupada para %s", remoteJID)
	} else {
		debounceMap[remoteJID] = &debounceEntry{
			message: content,
		}
	}

	// Reinicia timer
	debounceMap[remoteJID].timer = time.AfterFunc(time.Duration(debounceSeconds)*time.Second, func() {
		debounceMu.Lock()
		entry, exists := debounceMap[remoteJID]
		if exists {
			groupedMessage := entry.message
			delete(debounceMap, remoteJID)
			debounceMu.Unlock()

			log.Printf("[TYPEBOT] Debounce completo para %s: %s", remoteJID, groupedMessage)
			callback(groupedMessage)
		} else {
			debounceMu.Unlock()
		}
	})
}

// ==================== Helpers ====================

func extractField(content, fieldName string) string {
	pattern := regexp.MustCompile(fieldName + `: (.*?)(?:\n|$)`)
	if m := pattern.FindStringSubmatch(content); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func findWaitSeconds(actions []ClientSideAction, messageID string) int {
	for _, action := range actions {
		if action.LastBubbleBlockID == messageID && action.Wait != nil {
			return action.Wait.SecondsToWaitFor
		}
	}
	return 0
}

// StartTypebotManual inicia uma sessão manualmente via API /typebot/start
func StartTypebotManual(instanceID uuid.UUID, req StartTypebotRequest) (*SessionResponse, error) {
	var inst models.Instance
	if err := database.DB.First(&inst, instanceID).Error; err != nil {
		return nil, fmt.Errorf("instância não encontrada")
	}

	var cfg models.TypebotConfig

	if req.TypebotConfigID != nil {
		// Usa config existente
		if err := database.DB.First(&cfg, req.TypebotConfigID).Error; err != nil {
			return nil, fmt.Errorf("configuração Typebot não encontrada")
		}
	} else if req.URL != "" && req.Typebot != "" {
		// Cria config temporária a partir dos dados passados
		cfg = models.TypebotConfig{
			InstanceID:   instanceID,
			URL:          req.URL,
			Typebot:      req.Typebot,
			Enabled:      true,
			DelayMessage: 1000,
		}
	} else {
		return nil, fmt.Errorf("informe typebot_config_id ou (url + typebot)")
	}

	// Fecha sessão anterior se existir
	var existingSession models.TypebotSession
	if database.DB.Where("typebot_config_id = ? AND remote_jid = ? AND status = ?",
		cfg.ID, req.RemoteJID, models.TypebotSessionOpened).First(&existingSession).Error == nil {
		closeSession(&existingSession, false)
	}

	pushName := ""
	if req.PrefilledVariables != nil {
		if pn, ok := req.PrefilledVariables["pushName"].(string); ok {
			pushName = pn
		}
	}

	// Merge prefilled variables da config com as do request
	if cfg.PrefilledVariables != "" && req.PrefilledVariables == nil {
		req.PrefilledVariables = make(map[string]interface{})
		var custom map[string]interface{}
		if json.Unmarshal([]byte(cfg.PrefilledVariables), &custom) == nil {
			for k, v := range custom {
				req.PrefilledVariables[k] = v
			}
		}
	}

	session, typebotResp, err := createNewSession(inst, cfg, req.RemoteJID, pushName)
	if err != nil {
		return nil, err
	}

	// Busca EvoServer para enviar respostas
	var evoServer models.EvoServer
	if err := database.DB.First(&evoServer, inst.EvoServerID).Error; err != nil {
		return nil, fmt.Errorf("EvoServer não encontrado")
	}

	evoClient := evoclient.New(evoServer.URL, evoServer.APIKey)

	go sendTypebotResponses(evoClient, inst.EvoToken, req.RemoteJID, session, cfg, typebotResp)

	return &SessionResponse{
		ID:              session.ID,
		RemoteJID:       session.RemoteJID,
		PushName:        session.PushName,
		SessionID:       session.SessionID,
		Status:          string(session.Status),
		AwaitUser:       session.AwaitUser,
		TypebotConfigID: session.TypebotConfigID,
		CreatedAt:       session.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       session.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
