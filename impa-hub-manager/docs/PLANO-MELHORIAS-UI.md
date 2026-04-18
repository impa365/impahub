# Plano de Melhorias: ImpaHub Manager — UI/UX

**Data**: 17/04/2026  
**Status**: Aguardando aprovação  
**Avaliação atual**: 6.2/10

---

## 1. Estado Atual

### O que funciona bem
- Arquitetura limpa (React 19, Zustand, Zod, Vite)
- Responsividade básica (sidebar colapsível, grids adaptáveis)
- Dark mode implementado
- Login com branding split-view
- Validação de forms com Zod
- Toast notifications (Sonner)
- Ícones consistentes (Lucide)

### O que está ruim

| Problema | Onde | Impacto |
|----------|------|---------|
| Tabela de Usuários é HTML nativa sem paginação, sorting, filtros | `Users.tsx` | Crítico |
| Modals com scroll horrível em formulários longos | `Typebot.tsx`, `Chatwoot.tsx` | Alto |
| Sem componentes avançados: tabs, tooltips, popovers, breadcrumbs | Global | Alto |
| Loading é só um spinner girando — sem skeleton loaders | Todas as páginas | Médio |
| Dashboard raso: só stats + lista curta de instâncias | `Dashboard.tsx` | Médio |
| Sem tratamento visual de erros — catch silencioso | Todas as páginas | Médio |
| Sem paginação em nenhuma listagem | `Instances.tsx`, `Users.tsx` | Alto |
| Design "inacabado" — componentes parecem protótipo | Global | Alto |
| Sem gráficos/charts no dashboard | `Dashboard.tsx` | Médio |
| Z-index caótico (40, 50 misturados) | Modals, Sidebar | Baixo |

---

## 2. Plano de Implementação

### FASE 1 — Componentes Base Faltantes (fundação)

**Prioridade**: URGENTE

| # | Componente | Descrição |
|---|-----------|-----------|
| 1.1 | `Table.tsx` | Componente de tabela reutilizável com header sticky, hover rows, striping, responsive |
| 1.2 | `Pagination.tsx` | Componente de paginação (prev/next + page numbers + items per page) |
| 1.3 | `Tabs.tsx` | Tabs horizontais com variants (underline, pills, boxed) |
| 1.4 | `Tooltip.tsx` | Tooltip posicionado com delay e arrow (CSS puro ou floating-ui) |
| 1.5 | `Skeleton.tsx` | Skeleton loader animado (pulse) para cards, tabelas, listas |
| 1.6 | `Alert.tsx` | Alert box com variants: info, success, warning, error + ícone + dismissível |
| 1.7 | `Breadcrumbs.tsx` | Breadcrumbs automático baseado na rota atual |
| 1.8 | `EmptyState.tsx` | Componente padronizado de empty state (ícone + título + descrição + CTA) |
| 1.9 | `ConfirmDialog.tsx` | Dialog de confirmação reutilizável (substituir window.confirm) |
| 1.10 | `Dropdown.tsx` | Menu dropdown com items, separadores, ícones |

---

### FASE 2 — Refatorar Páginas Existentes

**Prioridade**: Alta

#### 2.1 — Tabela de Usuários (`Users.tsx`)
- Substituir `<table>` nativa pelo novo componente `Table`
- Adicionar: paginação (10/25/50 por página), sorting por coluna (clique no header), busca/filtro por nome/email, ações em lote (selecionar múltiplos)
- Skeleton loader enquanto carrega

#### 2.2 — Instâncias (`Instances.tsx`)
- Adicionar paginação no grid de cards
- Trocar polling de 5s por mecanismo mais suave (fade transition nos updates)
- Adicionar filtros: por status (conectada/desconectada/QR), por servidor
- Skeleton cards enquanto carrega

#### 2.3 — Chatwoot (`Chatwoot.tsx`)
- Usar `Tabs` para separar "Configuradas" vs "Não configuradas" (em vez de 2 seções)
- Formulário do modal: converter scroll overflow para stepper/wizard em forms longos
- Adicionar campo `ignore_jids` (novo campo adicionado no backend)
- Adicionar toggle `groups_ignore` com tooltip explicativo

#### 2.4 — Typebot (`Typebot.tsx`)
- Usar `Tabs` para separar instâncias/configs, sessions, settings
- Modals longas: converter para drawer lateral (slide-in) ou stepper
- Adicionar skeleton loader nas sessões

#### 2.5 — Servidores (`Servers.tsx`)
- Adicionar indicador visual de saúde do servidor (ping/latência)
- Status badge mais visível (pulsating dot para "ativo")

---

### FASE 3 — Dashboard Melhorado

**Prioridade**: Média

| # | Melhoria | Descrição |
|---|----------|-----------|
| 3.1 | Gráfico de atividade | Chart de mensagens processadas ao longo do tempo (7/30 dias) — instalar `recharts` ou `chart.js` |
| 3.2 | Status em tempo real | Cards de instância com indicador pulsante (verde/vermelho) |
| 3.3 | Últimas mensagens | Widget mostrando últimas mensagens processadas |
| 3.4 | Quick Actions | Botões de ação rápida: criar instância, conectar WhatsApp, ver logs |
| 3.5 | Alertas/Notificações | Banner de alertas no topo: instâncias desconectadas, erros recentes |
| 3.6 | Métricas de Chatwoot | Cards de mensagens enviadas/recebidas no dia |

---

### FASE 4 — Polish Visual (aparência profissional)

**Prioridade**: Média

| # | Melhoria | Descrição |
|---|----------|-----------|
| 4.1 | Skeleton loaders | Substituir TODOS os spinners por skeleton loaders (cards, tabelas, listas) |
| 4.2 | Transições de página | Fade suave entre páginas (framer-motion ou view transitions API) |
| 4.3 | Sidebar melhorada | Adicionar badge de contagem (ex: "3" instâncias desconectadas), separadores visuais entre seções, tooltip nos ícones quando colapsada |
| 4.4 | Cards com micro-interações | Hover lift sutil, click ripple |
| 4.5 | Error boundaries visuais | Componente de erro bonito em vez de crash/tela branca |
| 4.6 | Focus management | Focus rings customizados (primary color), keyboard navigation |
| 4.7 | Animações de entrada | Stagger animation nos grids de cards (cards aparecem um a um com delay) |
| 4.8 | Header melhorado | Adicionar breadcrumbs, notificações icon, link p/ perfil rápido |

---

### FASE 5 — Funcionalidades Novas

**Prioridade**: Baixa

| # | Feature | Descrição |
|---|---------|-----------|
| 5.1 | Logs em tempo real | Página de logs com WebSocket, filtro por instância/evento, auto-scroll |
| 5.2 | Webhook tester | Ferramenta para testar webhook manualmente (enviar payload fake) |
| 5.3 | Importação em massa | Upload CSV para criar instâncias/contatos em lote |
| 5.4 | Busca global | Cmd+K / Ctrl+K para buscar instâncias, servidores, configs rápido |
| 5.5 | Atalhos de teclado | Navegação por teclado entre páginas |
| 5.6 | Export/Import configs | Exportar/importar configurações de Chatwoot/Typebot em JSON |

---

## 3. Dependências a Instalar

```bash
# Prioridade 1 - Componentes
npm install @floating-ui/react     # Tooltips, Popovers, Dropdowns posicionados
npm install recharts               # Gráficos no Dashboard

# Prioridade 2 - Polish
npm install framer-motion          # Animações de página e componentes

# Prioridade 3 - Features
npm install cmdk                   # Command palette (Ctrl+K)
```

**NÃO instalar** UI libraries inteiras (MUI, Chakra, Ant). O approach custom com CVA + Tailwind é bom, só precisa mais componentes.

---

## 4. Resumo de Prioridades

| Prioridade | Fase | O que melhora |
|------------|------|---------------|
| 🔴 URGENTE | Fase 1 | Fundação: componentes base que faltam |
| 🟠 Alta | Fase 2 | Páginas existentes ficam usáveis e profissionais |
| 🟡 Média | Fase 3 | Dashboard deixa de ser raso |
| 🟡 Média | Fase 4 | Visual passa de "protótipo" para "produção" |
| 🟢 Baixa | Fase 5 | Features power-user |

---

## 5. Arquivos Impactados

```
src/
├── components/
│   └── ui/
│       ├── Table.tsx          ← NOVO (Fase 1)
│       ├── Pagination.tsx     ← NOVO (Fase 1)
│       ├── Tabs.tsx           ← NOVO (Fase 1)
│       ├── Tooltip.tsx        ← NOVO (Fase 1)
│       ├── Skeleton.tsx       ← NOVO (Fase 1)
│       ├── Alert.tsx          ← NOVO (Fase 1)
│       ├── Breadcrumbs.tsx    ← NOVO (Fase 1)
│       ├── EmptyState.tsx     ← NOVO (Fase 1)
│       ├── ConfirmDialog.tsx  ← NOVO (Fase 1)
│       ├── Dropdown.tsx       ← NOVO (Fase 1)
│       └── index.ts           ← ATUALIZAR exports
│
├── pages/
│   ├── Users.tsx              ← REFATORAR (Fase 2)
│   ├── Instances.tsx          ← REFATORAR (Fase 2)
│   ├── Chatwoot.tsx           ← REFATORAR (Fase 2)
│   ├── Typebot.tsx            ← REFATORAR (Fase 2)
│   ├── Servers.tsx            ← REFATORAR (Fase 2)
│   └── Dashboard.tsx          ← REFATORAR (Fase 3)
│
├── components/layout/
│   ├── Sidebar.tsx            ← MELHORAR (Fase 4)
│   └── Header.tsx             ← MELHORAR (Fase 4)
│
└── styles/
    └── globals.css             ← TOKENS (Fase 4)
```
