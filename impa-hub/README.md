# IMPA HUB - Integration Hub

Hub de integrações para conectar o **Evolution GO** com sistemas externos como **Chatwoot**, **Typebot** e outros.

## Arquitetura

```
┌─────────────────┐     ┌──────────────┐     ┌───────────────┐
│   WhatsApp      │────▶│ Evolution GO │────▶│   IMPA HUB    │
│   (usuários)    │◀────│   (API)      │◀────│ (integrador)  │
└─────────────────┘     └──────────────┘     └───────┬───────┘
                                                      │
                                              ┌───────┴───────┐
                                              │               │
                                        ┌─────▼─────┐  ┌─────▼─────┐
                                        │  Chatwoot  │  │  Typebot  │
                                        │  (suporte) │  │  (bots)   │
                                        └───────────┘  └───────────┘
```

### Fluxo de Mensagens

1. **WhatsApp → Chatwoot**: Mensagem chega no WhatsApp → Evolution GO envia webhook para IMPA HUB → IMPA HUB cria contato/conversa e envia mensagem para o Chatwoot
2. **Chatwoot → WhatsApp**: Agente responde no Chatwoot → Chatwoot envia webhook para IMPA HUB → IMPA HUB envia mensagem via Evolution GO API → Mensagem chega no WhatsApp

## Funcionalidades

### Multi-tenancy
- **SuperAdmin**: Gerencia todos os usuários, define quotas e permissões
- **Admin**: Gerencia suas instâncias e integrações
- **User**: Usuário comum com acesso limitado

### Quotas por Usuário
- Número máximo de instâncias WhatsApp
- Número máximo de conexões Chatwoot
- Número máximo de servidores Evolution GO
- Permissão para usar Chatwoot/Typebot (habilitado/desabilitado)

### Multi-servidor
- Conecte múltiplos servidores Evolution GO
- Cada usuário pode ter instâncias em servidores diferentes
- Teste de conexão com servidores

### Integração Chatwoot
- Criação automática de inbox no Chatwoot
- Criação automática de contatos e conversas
- Envio bidirecional de mensagens (texto e mídia)
- Configuração de assinatura de mensagens
- Reabertura automática de conversas
- Suporte a contatos brasileiros (merge 9 dígitos)
- Webhook configurado automaticamente

## API Endpoints

### Auth
| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/change-password` | Alterar senha |
| GET | `/api/v1/auth/me` | Dados do usuário logado |

### Admin (SuperAdmin only)
| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/v1/admin/users` | Listar usuários |
| POST | `/api/v1/admin/users` | Criar usuário |
| GET | `/api/v1/admin/users/:id` | Detalhes do usuário |
| PUT | `/api/v1/admin/users/:id` | Atualizar usuário |
| DELETE | `/api/v1/admin/users/:id` | Remover usuário |
| PUT | `/api/v1/admin/users/:id/quotas` | Atualizar quotas |
| POST | `/api/v1/admin/users/:id/reset-password` | Resetar senha |

### Servidores Evolution GO
| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/v1/servers` | Listar servidores |
| POST | `/api/v1/servers` | Adicionar servidor |
| GET | `/api/v1/servers/:id` | Detalhes do servidor |
| PUT | `/api/v1/servers/:id` | Atualizar servidor |
| DELETE | `/api/v1/servers/:id` | Remover servidor |
| POST | `/api/v1/servers/:id/test` | Testar conexão |

### Instâncias WhatsApp
| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/v1/instances` | Listar instâncias |
| POST | `/api/v1/instances` | Criar instância |
| GET | `/api/v1/instances/:id` | Detalhes da instância |
| DELETE | `/api/v1/instances/:id` | Remover instância |
| POST | `/api/v1/instances/:id/connect` | Conectar (gera QR) |
| GET | `/api/v1/instances/:id/status` | Status da conexão |
| GET | `/api/v1/instances/:id/qr` | QR Code |
| POST | `/api/v1/instances/:id/disconnect` | Desconectar |
| POST | `/api/v1/instances/:id/logout` | Logout |
| POST | `/api/v1/instances/:id/reconnect` | Reconectar |
| POST | `/api/v1/instances/:id/send/text` | Enviar texto |
| POST | `/api/v1/instances/:id/send/media` | Enviar mídia |

### Integração Chatwoot
| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/v1/integrations/chatwoot` | Listar configurações |
| POST | `/api/v1/integrations/chatwoot/set` | Configurar Chatwoot |
| GET | `/api/v1/integrations/chatwoot/:instanceId` | Detalhes da config |
| PUT | `/api/v1/integrations/chatwoot/:instanceId` | Atualizar config |
| DELETE | `/api/v1/integrations/chatwoot/:instanceId` | Remover config |

### Webhooks (Sem autenticação)
| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/webhook/:instanceId` | Webhook do Evolution GO |
| POST | `/chatwoot/webhook/:instanceId` | Webhook do Chatwoot |

## Setup Rápido

### Com Docker (recomendado)

```bash
cd impa-hub

# Copie e configure o .env
cp .env.example .env
# Edite o .env com suas configurações

# Suba os containers
docker compose up -d --build
```

### Local

```bash
cd impa-hub

# Configure o .env
cp .env.example .env

# Instale dependências
go mod tidy

# Execute
make dev
```

## Configuração

### Variáveis de Ambiente

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `SERVER_PORT` | Porta do servidor | `8080` |
| `BASE_URL` | URL pública do IMPA HUB | `http://localhost:8080` |
| `DATABASE_URL` | Connection string PostgreSQL | - |
| `JWT_SECRET` | Chave secreta para JWT **(obrigatório)** | - |
| `JWT_EXPIRATION_HOURS` | Expiração do token | `24` |
| `ADMIN_EMAIL` | Email do super admin | `admin@impa.hub` |
| `ADMIN_PASSWORD` | Senha do super admin | `admin123` |
| `LOG_LEVEL` | Nível de log | `info` |
| `CORS_ORIGINS` | Origens CORS permitidas | `*` |

### Primeiro Acesso

1. Após iniciar o servidor, faça login com as credenciais do super admin
2. Crie usuários e defina suas quotas
3. Adicione servidores Evolution GO
4. Crie instâncias WhatsApp
5. Configure integrações Chatwoot

## Exemplos de Uso

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@impa.hub", "password": "admin123"}'
```

### Adicionar Servidor EVO
```bash
curl -X POST http://localhost:8080/api/v1/servers \
  -H "Authorization: Bearer SEU_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Meu Servidor",
    "url": "http://meu-evo-go:4000",
    "apiKey": "CHAVE_API_DO_EVO_GO"
  }'
```

### Criar Instância
```bash
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Authorization: Bearer SEU_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "evoServerId": "ID_DO_SERVIDOR",
    "instanceName": "minha-instancia"
  }'
```

### Conectar Instância (gera QR Code)
```bash
curl -X POST http://localhost:8080/api/v1/instances/ID_DA_INSTANCIA/connect \
  -H "Authorization: Bearer SEU_TOKEN"
```

### Configurar Chatwoot
```bash
curl -X POST http://localhost:8080/api/v1/integrations/chatwoot/set \
  -H "Authorization: Bearer SEU_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instanceId": "ID_DA_INSTANCIA",
    "enabled": true,
    "url": "https://meu-chatwoot.com",
    "token": "TOKEN_DO_CHATWOOT",
    "accountId": "1",
    "nameInbox": "WhatsApp",
    "signMsg": true,
    "autoCreate": true,
    "reopenConversation": true,
    "conversationPending": false,
    "mergeBrazilContacts": true
  }'
```

## Estrutura do Projeto

```
impa-hub/
├── cmd/
│   └── impa-hub/
│       └── main.go              # Entry point
├── internal/
│   ├── admin/                   # Gestão de usuários (SuperAdmin)
│   │   ├── handler.go
│   │   └── service.go
│   ├── auth/                    # Autenticação JWT
│   │   ├── handler.go
│   │   └── service.go
│   ├── chatwoot/                # Integração Chatwoot completa
│   │   ├── client.go            # Client API Chatwoot
│   │   ├── handler.go           # Endpoints HTTP
│   │   ├── service.go           # Lógica de negócio
│   │   └── webhook_processor.go # Bridge EVO <-> Chatwoot
│   ├── config/                  # Configuração
│   │   └── config.go
│   ├── database/                # Conexão e migração
│   │   └── database.go
│   ├── evoclient/               # Client API Evolution GO
│   │   └── client.go
│   ├── instance/                # Gestão de instâncias
│   │   ├── handler.go
│   │   └── service.go
│   ├── middleware/               # Auth middleware
│   │   └── auth.go
│   ├── models/                  # Modelos do banco
│   │   └── models.go
│   ├── server/                  # Gestão de servidores EVO
│   │   ├── handler.go
│   │   └── service.go
│   └── webhook/                 # Receptor de webhooks EVO
│       └── handler.go
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── Makefile
└── .env.example
```

## Tecnologias

- **Go 1.22** - Linguagem principal
- **Gin** - Framework HTTP
- **GORM** - ORM para PostgreSQL
- **JWT** - Autenticação
- **PostgreSQL** - Banco de dados
- **Docker** - Containerização
