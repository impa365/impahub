import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { MessageSquare, Settings2, Trash2, Plus, ChevronRight, ExternalLink, Wifi, WifiOff } from 'lucide-react'
import { Button, Input, Card, CardContent, Badge, Switch, SkeletonCard, EmptyState, ConfirmDialog, SlideOver } from '@/components/ui'
import { chatwootApi, instancesApi } from '@/services/api'
import type { Instance, ChatwootConfig } from '@/types'

const chatwootSchema = z.object({
  account_id: z.string().min(1, 'Account ID é obrigatório'),
  token: z.string().min(1, 'Token é obrigatório'),
  url: z.string().url('URL inválida'),
  sign_delimiter: z.string().optional(),
  inbox_name: z.string().min(1, 'Nome do Inbox é obrigatório'),
})

type ChatwootForm = z.infer<typeof chatwootSchema>

interface InstanceConfig {
  instance: Instance
  config: ChatwootConfig | null
}

export default function ChatwootPage() {
  const [data, setData] = useState<InstanceConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [panelOpen, setPanelOpen] = useState(false)
  const [selectedInstance, setSelectedInstance] = useState<Instance | null>(null)
  const [selectedConfig, setSelectedConfig] = useState<ChatwootConfig | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [enabled, setEnabled] = useState(true)
  const [signMsg, setSignMsg] = useState(true)
  const [reopenConv, setReopenConv] = useState(true)
  const [convPending, setConvPending] = useState(false)
  const [autoCreate, setAutoCreate] = useState(true)
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [instanceToDelete, setInstanceToDelete] = useState<string | null>(null)
  const [groupsIgnore, setGroupsIgnore] = useState(true)
  const [ignoreJids, setIgnoreJids] = useState('')

  const { register, handleSubmit, reset, formState: { errors } } = useForm<ChatwootForm>({
    resolver: zodResolver(chatwootSchema),
  })

  const loadData = async () => {
    try {
      const instances = await instancesApi.list()
      const configs = await Promise.all(
        instances.map(async (inst) => {
          try {
            const config = await chatwootApi.get(inst.id)
            return { instance: inst, config }
          } catch {
            return { instance: inst, config: null }
          }
        })
      )
      setData(configs)
    } catch {
      toast.error('Erro ao carregar dados')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadData() }, [])

  const openConfig = (inst: Instance, config: ChatwootConfig | null) => {
    setSelectedInstance(inst)
    setSelectedConfig(config)
    if (config) {
      reset({
        account_id: String(config.account_id),
        token: config.token,
        url: config.url,
        sign_delimiter: config.sign_delimiter || '\\n',
        inbox_name: config.inbox_name || '',
      })
      setEnabled(config.enabled ?? config.is_active ?? true)
      setSignMsg(config.sign_msg)
      setReopenConv(config.reopen_conversation)
      setConvPending(config.conversation_pending)
      setAutoCreate(config.auto_create)
      setGroupsIgnore(config.groups_ignore ?? true)
      setIgnoreJids(config.ignore_jids ?? '')
    } else {
      reset({ account_id: '', token: '', url: '', sign_delimiter: '\\n', inbox_name: '' })
      setEnabled(true)
      setSignMsg(true)
      setReopenConv(true)
      setConvPending(false)
      setAutoCreate(true)
      setGroupsIgnore(true)
      setIgnoreJids('')
    }
    setPanelOpen(true)
  }

  const onSubmit = async (formData: ChatwootForm) => {
    if (!selectedInstance) return
    setSubmitting(true)
    try {
      await chatwootApi.set({
        instance_id: selectedInstance.id,
        ...formData,
        enabled: enabled,
        sign_msg: signMsg,
        reopen_conversation: reopenConv,
        conversation_pending: convPending,
        auto_create: autoCreate,
        groups_ignore: groupsIgnore,
        ignore_jids: ignoreJids,
      })
      toast.success('Chatwoot configurado!')
      setPanelOpen(false)
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao configurar')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (instanceId: string) => {
    setInstanceToDelete(instanceId)
    setDeleteConfirmOpen(true)
  }

  const confirmDelete = async () => {
    if (!instanceToDelete) return
    try {
      await chatwootApi.delete(instanceToDelete)
      toast.success('Integração removida!')
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao remover')
    } finally {
      setDeleteConfirmOpen(false)
      setInstanceToDelete(null)
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Integração Chatwoot</h1>
          <p className="text-muted-foreground mt-1">Configure o Chatwoot para suas instâncias WhatsApp</p>
        </div>
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <SkeletonCard key={i} />
          ))}
        </div>
      </div>
    )
  }

  const configured = data.filter(d => d.config)
  const notConfigured = data.filter(d => !d.config)

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Integração Chatwoot</h1>
          <p className="text-muted-foreground mt-1">Configure o Chatwoot para suas instâncias WhatsApp</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-1.5 rounded-md bg-muted px-2.5 py-1.5 text-xs font-medium text-foreground">
            <Wifi className="h-3 w-3" /> {configured.length} configurada{configured.length !== 1 ? 's' : ''}
          </span>
          <span className="inline-flex items-center gap-1.5 rounded-md bg-muted px-2.5 py-1.5 text-xs font-medium text-muted-foreground">
            <WifiOff className="h-3 w-3" /> {notConfigured.length} pendente{notConfigured.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      {data.length === 0 ? (
        <EmptyState
          icon={MessageSquare}
          title="Nenhuma instância"
          description="Crie instâncias primeiro para configurar o Chatwoot"
        />
      ) : (
        <div className="space-y-2">
          {data.map(({ instance, config }) => {
            const isConfigured = !!config
            const isActive = config?.is_active ?? false

            return (
              <div
                key={instance.id}
                onClick={() => openConfig(instance, config)}
                className="group flex items-center gap-4 rounded-xl border border-border bg-card px-5 py-4 cursor-pointer hover:bg-white/5 transition-all duration-200 hover-lift"
              >
                {/* Status indicator */}
                <div className={`h-9 w-9 rounded-md flex items-center justify-center shrink-0 bg-muted`}>
                  <MessageSquare className={`h-4 w-4 ${isConfigured ? 'text-foreground' : 'text-muted-foreground'}`} />
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="font-semibold text-sm truncate">{instance.instance_name}</p>
                    {isConfigured && (
                      <Badge variant={isActive ? 'success' : 'secondary'} className="text-[10px] shrink-0">
                        {isActive ? 'Ativo' : 'Inativo'}
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5 truncate">
                    {isConfigured
                      ? <><span className="font-mono">{config!.url}</span> · Account #{config!.account_id}</>
                      : <span className="text-muted-foreground/60">Sem configuração — clique para configurarr</span>
                    }
                  </p>
                </div>

                {/* Feature badges */}
                {isConfigured && (
                  <div className="hidden md:flex items-center gap-1.5 shrink-0">
                    {config!.sign_msg && <Badge variant="outline" className="text-[10px]">Sign</Badge>}
                    {config!.reopen_conversation && <Badge variant="outline" className="text-[10px]">Reabrir</Badge>}
                    {config!.auto_create && <Badge variant="outline" className="text-[10px]">Auto</Badge>}
                  </div>
                )}

                {/* Actions */}
                <div className="flex items-center gap-1 shrink-0">
                  {isConfigured && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => { e.stopPropagation(); handleDelete(instance.id) }}
                      className="text-destructive hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                      title="Remover"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  )}
                  <ChevronRight className="h-4 w-4 text-muted-foreground/40 group-hover:text-muted-foreground transition-colors" />
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Config SlideOver Panel */}
      <SlideOver
        open={panelOpen}
        onClose={() => setPanelOpen(false)}
        title={selectedConfig ? 'Editar Configuração' : 'Nova Configuração'}
        description={selectedInstance?.instance_name}
        size="lg"
        footer={
          <div className="flex justify-between">
            <Button type="button" variant="outline" onClick={() => setPanelOpen(false)}>Cancelar</Button>
            <Button onClick={handleSubmit(onSubmit)} loading={submitting}>Salvar Configuração</Button>
          </div>
        }
      >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          {/* Connection */}
          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Conexão</h3>
            <div className="space-y-3">
              <Input id="url" label="URL do Chatwoot" placeholder="https://chatwoot.example.com" error={errors.url?.message} {...register('url')} />
              <Input id="token" label="Token da API" placeholder="token..." error={errors.token?.message} {...register('token')} />
              <div className="grid grid-cols-2 gap-3">
                <Input id="account_id" label="Account ID" error={errors.account_id?.message} {...register('account_id')} />
                <Input id="inbox_name" label="Nome do Inbox" placeholder="WhatsApp" error={errors.inbox_name?.message} {...register('inbox_name')} />
              </div>
              <Input id="sign_delimiter" label="Delimitador de Assinatura" placeholder="\\n" {...register('sign_delimiter')} />
            </div>
          </section>

          {/* Options */}
          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Opções</h3>
            <div className="space-y-3 rounded-lg bg-muted p-4">
              <Switch checked={enabled} onChange={setEnabled} label="Ativado" />
              <Switch checked={signMsg} onChange={setSignMsg} label="Assinar mensagens (sign_msg)" />
              <Switch checked={reopenConv} onChange={setReopenConv} label="Reabrir conversas" />
              <Switch checked={convPending} onChange={setConvPending} label="Conversas como pendentes" />
              <Switch checked={autoCreate} onChange={setAutoCreate} label="Auto-criar inbox/contatos" />
            </div>
          </section>

          {/* Group filters */}
          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Filtros de Grupo</h3>
            <div className="space-y-3 rounded-lg bg-muted p-4">
              <Switch checked={groupsIgnore} onChange={setGroupsIgnore} label="Ignorar mensagens de grupos" />
              <div>
                <label className="text-sm font-medium text-foreground block mb-1.5">JIDs para ignorar</label>
                <textarea
                  value={ignoreJids}
                  onChange={(e) => setIgnoreJids(e.target.value)}
                  placeholder="Um JID por linha (ex: 5511999999999@s.whatsapp.net)"
                  rows={3}
                  className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1 focus:ring-offset-background resize-none font-mono"
                />
                <p className="text-[11px] text-muted-foreground mt-1">Números ou JIDs que serão ignorados pelo bot</p>
              </div>
            </div>
          </section>
        </form>
      </SlideOver>

      {/* Delete Confirm Dialog */}
      <ConfirmDialog
        open={deleteConfirmOpen}
        onClose={() => { setDeleteConfirmOpen(false); setInstanceToDelete(null) }}
        onConfirm={confirmDelete}
        title="Remover Integração"
        description="Tem certeza que deseja remover a integração Chatwoot desta instância?"
        confirmLabel="Remover"
        variant="danger"
      />
    </div>
  )
}
