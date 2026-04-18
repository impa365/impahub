# Plano de Implementação: Tratamento de Mensagens de Grupo no ImpaHub

**Data**: 17/04/2026  
**Status**: Aguardando aprovação  
**Prioridade**: URGENTE

---

## 1. Diagnóstico do Bug

Mensagens de grupo (`@g.us`) estão sendo processadas como mensagens diretas (`@s.whatsapp.net`), causando:

- Criação de contatos fantasma no Chatwoot com IDs de grupo como "telefone"
- Conversas inválidas associadas a grupos em vez de pessoas
- Sessões de Typebot disparadas por mensagens de grupo

### 1.1 Chatwoot — `internal/chatwoot/webhook_processor.go`

**Linha ~231**: O único filtro existente só bloqueia `@broadcast`. Grupos passam direto:

```go
// ÚNICO filtro existente — grupos NÃO são filtrados
if strings.HasSuffix(msgData.Info.Chat, "@broadcast") || msgData.Info.Chat == "status@broadcast" {
    return
}
```

**Linha ~1023**: A função `cleanJID()` remove o sufixo `@g.us` cegamente, transformando o ID do grupo em "telefone":

```go
func cleanJID(jid string) string {
    jid = strings.Split(jid, "@")[0]  // "120363123456789@g.us" → "120363123456789" ← INVÁLIDO
    jid = strings.Split(jid, ":")[0]
    return jid
}
```

**Resultado**: `120363123456789@g.us` vira contato com telefone `120363123456789`.

### 1.2 Typebot — `internal/typebot/webhook_processor.go`

**Linha ~89**: Só filtra `status@broadcast`, grupos passam:

```go
if msgData.Info.Chat == "status@broadcast" {
    return
}
```

**Linha ~240**: O `isIgnoredJID()` até suporta wildcard `@g.us`, mas **depende de configuração manual** por bot. Sem filtro padrão.

### 1.3 Model ChatwootConfig — `internal/models/models.go`

O `ChatwootConfig` **não possui** campo `IgnoreJids`, ao contrário do `TypebotConfig` e `TypebotSetting` que já têm.

---

## 2. Como a Evolution API trata (referência)

Arquivo: `src/api/integrations/chatbot/chatwoot/services/chatwoot.service.ts`

| Funcionalidade | Implementação |
|----------------|---------------|
| Detecta grupo | `isGroup = remoteJid.endsWith('@g.us')` |
| Sistema ignoreJids | Filtra por `@g.us`, `@s.whatsapp.net` ou JIDs específicos |
| Metadata do grupo | Busca nome e participantes via `groupMetadata()` |
| Contato do grupo | Cria com nome `[NomeGrupo] (GROUP)` |
| Contato do participante | Cria separadamente usando `key.participant` |
| Formatação | Prefixa: `**+55 11 99999-9999 - João Silva:**\n\nconteúdo` |

---

## 3. Plano de Implementação

### FASE 1 — Filtragem Imediata (corrigir o bug)

**Prioridade**: URGENTE  
**Estimativa**: Impacto baixo, risco baixo

| # | Arquivo | Tarefa |
|---|---------|--------|
| 1.1 | `internal/chatwoot/webhook_processor.go` | Criar função `isGroup(jid string) bool` → `strings.HasSuffix(jid, "@g.us")` |
| 1.2 | `internal/chatwoot/webhook_processor.go` | Na `processIncomingMessage()`, após filtro de broadcast (linha ~231), adicionar: `if isGroup(msgData.Info.Chat) { return }` |
| 1.3 | `internal/typebot/webhook_processor.go` | Criar mesma função `isGroup()` e adicionar filtro padrão após linha 89 |
| 1.4 | `internal/models/models.go` | Adicionar campo `IgnoreJids string` no `ChatwootConfig` |

**Resultado**: Grupos param de criar contatos fantasma. Mensagens de grupo são descartadas por padrão.

---

### FASE 2 — Suporte a Mensagens de Grupo no Chatwoot

**Prioridade**: Alta  
**Dependência**: Fase 1

| # | Arquivo | Tarefa |
|---|---------|--------|
| 2.1 | `internal/models/models.go` | Adicionar no `ChatwootConfig`: `GroupsIgnore bool` (default `true`), `GroupNamePrefix string` (default `" (GROUP)"`) |
| 2.2 | `internal/chatwoot/webhook_processor.go` | Refatorar `processIncomingMessage()`: se `isGroup()` e `!cfg.GroupsIgnore`, chamar nova função `processGroupMessage()` |
| 2.3 | `internal/chatwoot/webhook_processor.go` | Criar `processGroupMessage()` que: |
|     |                                          | — Extrai `participant` do campo `Info.Sender` (quem mandou no grupo) |
|     |                                          | — Usa `Info.Chat` para identificar o grupo |
|     |                                          | — Cria/busca contato do GRUPO (com Chat JID) |
|     |                                          | — Cria/busca contato do PARTICIPANTE (com Sender JID) |
|     |                                          | — Formata mensagem: `**+número - NomePessoa:**\n\nconteúdo` |
| 2.4 | `internal/chatwoot/types.go` | Confirmar structs — `EvoMessageInfo` já tem `Sender` e `Chat` separados ✅ |
| 2.5 | `internal/evoclient/` | Criar método `GetGroupMetadata(groupJID)` para buscar nome do grupo via Evolution GO API |

**Resultado**: Mensagens de grupo aparecem corretamente no Chatwoot com nome do grupo e identificação do participante.

---

### FASE 3 — Sistema de IgnoreJids no Chatwoot (configurável)

**Prioridade**: Média  
**Dependência**: Fase 1

| # | Arquivo | Tarefa |
|---|---------|--------|
| 3.1 | `internal/chatwoot/webhook_processor.go` | Implementar `isIgnoredJID()` para Chatwoot (reutilizar lógica do Typebot com wildcards) |
| 3.2 | `internal/chatwoot/webhook_processor.go` | Na `processIncomingMessage()`, verificar `ignoreJids` do config antes de processar |
| 3.3 | Handlers da API admin | Expor campo `ignore_jids` na API de configuração do Chatwoot |
| 3.4 | `impa-hub-manager/` (frontend) | Adicionar campo "JIDs a ignorar" na tela de configuração Chatwoot |

**Resultado**: Controle granular de quais JIDs processar/ignorar no Chatwoot.

---

### FASE 4 — Typebot: Filtro Padrão de Grupos

**Prioridade**: Baixa  
**Dependência**: Fase 1

| # | Arquivo | Tarefa |
|---|---------|--------|
| 4.1 | `internal/typebot/webhook_processor.go` | Adicionar filtro `isGroup()` antes do `isIgnoredJID()` para ignorar grupos por padrão |
| 4.2 | `internal/models/models.go` | Adicionar `AllowGroups bool` no `TypebotConfig` (default `false`) e `TypebotSetting` |
| 4.3 | `internal/typebot/webhook_processor.go` | Só processar grupo se `AllowGroups == true` no config do bot ou settings globais |

**Resultado**: Typebot ignora grupos por padrão, com opção de habilitar por bot.

---

## 4. Resumo de Prioridades

| Prioridade | Fase | Descrição | Impacto |
|------------|------|-----------|---------|
| 🔴 URGENTE | Fase 1 | Filtrar grupos imediatamente | Para contatos fantasma |
| 🟠 Alta | Fase 2 | Suporte completo a grupos no Chatwoot | Grupos funcionam corretamente |
| 🟡 Média | Fase 3 | Sistema ignoreJids configurável no Chatwoot | Controle granular |
| 🟢 Baixa | Fase 4 | Filtro padrão no Typebot | Typebot protegido |

---

## 5. Arquivos Impactados

```
internal/
├── chatwoot/
│   └── webhook_processor.go   ← Fases 1, 2, 3
├── typebot/
│   └── webhook_processor.go   ← Fases 1, 4
├── models/
│   └── models.go              ← Fases 1, 2, 4
└── evoclient/
    └── (novo método)          ← Fase 2

impa-hub-manager/
└── (tela de config)           ← Fase 3
```

---

## 6. Referência: Formatos de JID do WhatsApp

| Tipo | Formato | Exemplo |
|------|---------|---------|
| Direto | `número@s.whatsapp.net` | `5511999999999@s.whatsapp.net` |
| Grupo | `id@g.us` | `120363123456789-1234567890@g.us` |
| Broadcast | `status@broadcast` | `status@broadcast` |
