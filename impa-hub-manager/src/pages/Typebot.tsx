import { useEffect, useState, useCallback } from 'react'
import { toast } from 'sonner'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Bot, Settings2, Trash2, Plus, Play, Pause, Square, Users2, Copy, ChevronDown, ChevronUp } from 'lucide-react'
import { Button, Input, Card, CardContent, CardHeader, CardTitle, SlideOver, Badge, Switch, Select } from '@/components/ui'
import { typebotApi, instancesApi } from '@/services/api'
import type { Instance, TypebotConfig, TypebotSession, TypebotSetting } from '@/types'

const typebotSchema = z.object({
  url: z.string().url('URL inválida'),
  typebot: z.string().min(1, 'ID do Typebot é obrigatório'),
  description: z.string().optional(),
  trigger_type: z.string(),
  trigger_operator: z.string(),
  trigger_value: z.string(),
  expire: z.coerce.number().min(0),
  keyword_finish: z.string(),
  delay_message: z.coerce.number().min(0),
  unknown_message: z.string(),
  debounce_time: z.coerce.number().min(0),
})

type TypebotForm = z.infer<typeof typebotSchema>

const settingsSchema = z.object({
  expire: z.coerce.number().min(0),
  keyword_finish: z.string(),
  delay_message: z.coerce.number().min(0),
  unknown_message: z.string(),
  debounce_time: z.coerce.number().min(0),
})

type SettingsForm = z.infer<typeof settingsSchema>

interface InstanceData {
  instance: Instance
  configs: TypebotConfig[]
}

export default function TypebotPage() {
  const [data, setData] = useState<InstanceData[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedInstances, setExpandedInstances] = useState<Set<string>>(new Set())

  // Config modal
  const [configModalOpen, setConfigModalOpen] = useState(false)
  const [selectedInstance, setSelectedInstance] = useState<Instance | null>(null)
  const [editingConfig, setEditingConfig] = useState<TypebotConfig | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [listeningFromMe, setListeningFromMe] = useState(false)
  const [stopBotFromMe, setStopBotFromMe] = useState(false)
  const [keepOpen, setKeepOpen] = useState(false)
  const [enabled, setEnabled] = useState(true)

  // Settings modal
  const [settingsModalOpen, setSettingsModalOpen] = useState(false)
  const [settingsInstance, setSettingsInstance] = useState<Instance | null>(null)
  const [currentSettings, setCurrentSettings] = useState<TypebotSetting | null>(null)
  const [settingsListeningFromMe, setSettingsListeningFromMe] = useState(false)
  const [settingsStopBotFromMe, setSettingsStopBotFromMe] = useState(false)
  const [settingsKeepOpen, setSettingsKeepOpen] = useState(false)
  const [settingsSubmitting, setSettingsSubmitting] = useState(false)

  // Sessions modal
  const [sessionsOpen, setSessionsOpen] = useState(false)
  const [sessionsInstance, setSessionsInstance] = useState<Instance | null>(null)
  const [sessions, setSessions] = useState<TypebotSession[]>([])

  const { register, handleSubmit, reset, watch, formState: { errors } } = useForm<TypebotForm>({
    resolver: zodResolver(typebotSchema),
  })

  const { register: registerSettings, handleSubmit: handleSubmitSettings, reset: resetSettings } = useForm<SettingsForm>({
    resolver: zodResolver(settingsSchema),
  })

  const triggerType = watch('trigger_type')

  const loadData = useCallback(async () => {
    try {
      const [instances, allConfigs] = await Promise.all([
        instancesApi.list(),
        typebotApi.findAll(),
      ])
      const grouped = instances.map(inst => ({
        instance: inst,
        configs: allConfigs.filter(c => c.instance_id === inst.id),
      }))
      setData(grouped)
      // Expand instances that have configs
      const withConfigs = new Set(grouped.filter(g => g.configs.length > 0).map(g => g.instance.id))
      setExpandedInstances(prev => new Set([...prev, ...withConfigs]))
    } catch {
      toast.error('Erro ao carregar dados')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadData() }, [loadData])

  const toggleExpand = (id: string) => {
    setExpandedInstances(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  // ========== Config CRUD ==========

  const openCreateConfig = (inst: Instance) => {
    setSelectedInstance(inst)
    setEditingConfig(null)
    reset({
      url: '',
      typebot: '',
      description: '',
      trigger_type: 'keyword',
      trigger_operator: 'contains',
      trigger_value: '',
      expire: 0,
      keyword_finish: '#sair',
      delay_message: 1000,
      unknown_message: '',
      debounce_time: 0,
    })
    setListeningFromMe(false)
    setStopBotFromMe(false)
    setKeepOpen(false)
    setEnabled(true)
    setConfigModalOpen(true)
  }

  const openEditConfig = (inst: Instance, config: TypebotConfig) => {
    setSelectedInstance(inst)
    setEditingConfig(config)
    reset({
      url: config.url,
      typebot: config.typebot,
      description: config.description || '',
      trigger_type: config.trigger_type,
      trigger_operator: config.trigger_operator,
      trigger_value: config.trigger_value || '',
      expire: config.expire,
      keyword_finish: config.keyword_finish || '',
      delay_message: config.delay_message,
      unknown_message: config.unknown_message || '',
      debounce_time: config.debounce_time,
    })
    setListeningFromMe(config.listening_from_me)
    setStopBotFromMe(config.stop_bot_from_me)
    setKeepOpen(config.keep_open)
    setEnabled(config.enabled)
    setConfigModalOpen(true)
  }

  const onSubmitConfig = async (formData: TypebotForm) => {
    if (!selectedInstance) return
    setSubmitting(true)
    try {
      if (editingConfig) {
        await typebotApi.update(editingConfig.id, {
          enabled,
          ...formData,
          listening_from_me: listeningFromMe,
          stop_bot_from_me: stopBotFromMe,
          keep_open: keepOpen,
        })
        toast.success('Typebot atualizado!')
      } else {
        await typebotApi.create({
          instance_id: selectedInstance.id,
          enabled,
          ...formData,
          listening_from_me: listeningFromMe,
          stop_bot_from_me: stopBotFromMe,
          keep_open: keepOpen,
        })
        toast.success('Typebot criado!')
      }
      setConfigModalOpen(false)
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao salvar')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDeleteConfig = async (typebotId: string) => {
    if (!confirm('Tem certeza que deseja remover este Typebot?')) return
    try {
      await typebotApi.delete(typebotId)
      toast.success('Typebot removido!')
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao remover')
    }
  }

  // ========== Settings ==========

  const openSettings = async (inst: Instance) => {
    setSettingsInstance(inst)
    try {
      const settings = await typebotApi.fetchSettings(inst.id)
      setCurrentSettings(settings)
      resetSettings({
        expire: settings?.expire || 0,
        keyword_finish: settings?.keyword_finish || '',
        delay_message: settings?.delay_message || 1000,
        unknown_message: settings?.unknown_message || '',
        debounce_time: settings?.debounce_time || 0,
      })
      setSettingsListeningFromMe(settings?.listening_from_me || false)
      setSettingsStopBotFromMe(settings?.stop_bot_from_me || false)
      setSettingsKeepOpen(settings?.keep_open || false)
      setSettingsModalOpen(true)
    } catch {
      toast.error('Erro ao carregar settings')
    }
  }

  const onSubmitSettings = async (formData: SettingsForm) => {
    if (!settingsInstance) return
    setSettingsSubmitting(true)
    try {
      await typebotApi.setSettings(settingsInstance.id, {
        ...formData,
        listening_from_me: settingsListeningFromMe,
        stop_bot_from_me: settingsStopBotFromMe,
        keep_open: settingsKeepOpen,
        typebot_id_fallback: currentSettings?.typebot_id_fallback,
      })
      toast.success('Settings salvas!')
      setSettingsModalOpen(false)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao salvar settings')
    } finally {
      setSettingsSubmitting(false)
    }
  }

  // ========== Sessions ==========

  const openSessions = async (inst: Instance) => {
    setSessionsInstance(inst)
    try {
      const data = await typebotApi.sessions(inst.id)
      setSessions(data || [])
      setSessionsOpen(true)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao carregar sessões')
    }
  }

  const handleChangeStatus = async (remoteJid: string, status: 'opened' | 'closed' | 'paused') => {
    if (!sessionsInstance) return
    try {
      await typebotApi.changeStatus(sessionsInstance.id, { remote_jid: remoteJid, status })
      toast.success(`Sessão ${status === 'closed' ? 'encerrada' : status === 'paused' ? 'pausada' : 'reaberta'}`)
      const updated = await typebotApi.sessions(sessionsInstance.id)
      setSessions(updated || [])
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao alterar status')
    }
  }

  // ========== Helpers ==========

  const triggerTypeLabel = (t: string) => {
    switch (t) {
      case 'all': return 'Todas mensagens'
      case 'keyword': return 'Palavra-chave'
      case 'none': return 'Desativado'
      case 'advanced': return 'Avançado (Regex)'
      default: return t
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    )
  }

  const withConfigs = data.filter(d => d.configs.length > 0)
  const withoutConfigs = data.filter(d => d.configs.length === 0)

  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Integração Typebot</h1>
        <p className="text-muted-foreground mt-1">Configure múltiplos fluxos Typebot por instância WhatsApp</p>
      </div>

      {/* Instances with configs */}
      {withConfigs.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Configuradas ({withConfigs.length})</h2>
          {withConfigs.map(({ instance, configs }) => (
            <Card key={instance.id} className="transition-all duration-200 hover-lift">
              <CardHeader className="pb-2 cursor-pointer" onClick={() => toggleExpand(instance.id)}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="h-9 w-9 rounded-md bg-muted flex items-center justify-center">
                      <Bot className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <CardTitle className="text-sm font-medium">{instance.instance_name}</CardTitle>
                      <p className="text-xs text-muted-foreground">{configs.length} bot{configs.length !== 1 ? 's' : ''} configurado{configs.length !== 1 ? 's' : ''}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button variant="ghost" size="sm" onClick={(e) => { e.stopPropagation(); openSettings(instance) }} title="Settings globais">
                      <Settings2 className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="sm" onClick={(e) => { e.stopPropagation(); openSessions(instance) }} title="Sessões">
                      <Users2 className="h-4 w-4" />
                    </Button>
                    <Button variant="outline" size="sm" onClick={(e) => { e.stopPropagation(); openCreateConfig(instance) }}>
                      <Plus className="h-3.5 w-3.5" /> Novo Bot
                    </Button>
                    {expandedInstances.has(instance.id) ? <ChevronUp className="h-4 w-4 text-muted-foreground" /> : <ChevronDown className="h-4 w-4 text-muted-foreground" />}
                  </div>
                </div>
              </CardHeader>
              {expandedInstances.has(instance.id) && (
                <CardContent className="pt-0">
                  <div className="space-y-2">
                    {configs.map(config => (
                      <div key={config.id} className="flex items-center justify-between rounded-lg bg-muted p-3.5 hover:bg-muted/70 transition-colors">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <p className="text-sm font-medium truncate">{config.description || config.typebot}</p>
                            <Badge variant={config.enabled ? 'success' : 'secondary'} className="text-[10px] shrink-0">
                              {config.enabled ? 'Ativo' : 'Inativo'}
                            </Badge>
                          </div>
                          <div className="flex items-center gap-2 text-xs text-muted-foreground">
                            <span className="truncate">{config.url}</span>
                            <span>&middot;</span>
                            <Badge variant="outline" className="text-[10px]">{triggerTypeLabel(config.trigger_type)}</Badge>
                            {config.trigger_value && <Badge variant="outline" className="text-[10px]">{config.trigger_value}</Badge>}
                          </div>
                        </div>
                        <div className="flex items-center gap-1 ml-2 shrink-0">
                          <Button variant="ghost" size="sm" onClick={() => openEditConfig(instance, config)} title="Editar">
                            <Settings2 className="h-3.5 w-3.5" />
                          </Button>
                          <Button variant="ghost" size="sm" onClick={() => { navigator.clipboard.writeText(config.id); toast.success('ID copiado!') }} title="Copiar ID">
                            <Copy className="h-3.5 w-3.5" />
                          </Button>
                          <Button variant="ghost" size="sm" onClick={() => handleDeleteConfig(config.id)} className="text-destructive hover:text-destructive" title="Remover">
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              )}
            </Card>
          ))}
        </div>
      )}

      {/* Instances without configs */}
      {withoutConfigs.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Sem Configuração ({withoutConfigs.length})</h2>
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {withoutConfigs.map(({ instance }) => (
              <Card key={instance.id} className="border-dashed border-2 shadow-none hover:border-border transition-colors">
                <CardContent className="flex items-center justify-between py-4">
                  <div className="flex items-center gap-3">
                    <div className="h-9 w-9 rounded-md bg-muted flex items-center justify-center">
                      <Bot className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="font-semibold text-sm">{instance.instance_name}</p>
                      <p className="text-xs text-muted-foreground">{instance.server_name}</p>
                    </div>
                  </div>
                  <Button variant="outline" size="sm" onClick={() => openCreateConfig(instance)}>
                    <Plus className="h-3.5 w-3.5" /> Configurar
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}

      {data.length === 0 && (
        <Card className="border-dashed border-2 shadow-none">
          <CardContent className="py-16 text-center">
            <div className="mx-auto w-12 h-12 rounded-md bg-muted flex items-center justify-center mb-4">
              <Bot className="h-6 w-6 text-muted-foreground" />
            </div>
            <h3 className="text-base font-medium mb-1.5">Nenhuma instância</h3>
            <p className="text-muted-foreground text-sm">Crie instâncias primeiro para configurar o Typebot</p>
          </CardContent>
        </Card>
      )}

      {/* Config SlideOver (Create/Edit) */}
      <SlideOver
        open={configModalOpen}
        onClose={() => setConfigModalOpen(false)}
        title={editingConfig ? 'Editar Typebot' : 'Novo Typebot'}
        description={selectedInstance?.instance_name}
        size="lg"
        footer={
          <div className="flex justify-between">
            <Button type="button" variant="outline" onClick={() => setConfigModalOpen(false)}>Cancelar</Button>
            <Button onClick={handleSubmit(onSubmitConfig)} loading={submitting}>{editingConfig ? 'Atualizar' : 'Criar'}</Button>
          </div>
        }
      >
        <form onSubmit={handleSubmit(onSubmitConfig)} className="space-y-6">
          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Identificação</h3>
            <div className="space-y-3">
              <Input id="description" label="Descrição (opcional)" placeholder="Bot de atendimento principal" {...register('description')} />
              <Input id="url" label="URL do Typebot" placeholder="https://typebot.example.com" error={errors.url?.message} {...register('url')} />
              <Input id="typebot" label="ID do Typebot (Public ID)" placeholder="meu-typebot-abc123" error={errors.typebot?.message} {...register('typebot')} />
            </div>
          </section>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Trigger</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <Select
                  id="trigger_type"
                  label="Tipo de Trigger"
                  options={[
                    { value: 'all', label: 'Todas as mensagens' },
                    { value: 'keyword', label: 'Palavra-chave' },
                    { value: 'none', label: 'Desativado (só manual)' },
                    { value: 'advanced', label: 'Avançado (Regex)' },
                  ]}
                  {...register('trigger_type')}
                />
                {(triggerType === 'keyword' || triggerType === 'advanced') && (
                  <Select
                    id="trigger_operator"
                    label="Operador"
                    options={[
                      { value: 'contains', label: 'Contém' },
                      { value: 'equals', label: 'Igual a' },
                      { value: 'startsWith', label: 'Começa com' },
                      { value: 'endsWith', label: 'Termina com' },
                      { value: 'regex', label: 'Regex' },
                    ]}
                    {...register('trigger_operator')}
                  />
                )}
              </div>
              {(triggerType === 'keyword' || triggerType === 'advanced') && (
                <Input id="trigger_value" label="Valor do Trigger" placeholder="oi, olá, menu..." {...register('trigger_value')} />
              )}
            </div>
          </section>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Tempos</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-3 gap-3">
                <Input id="expire" label="Expirar (min)" type="number" placeholder="0" {...register('expire')} />
                <Input id="delay_message" label="Delay (ms)" type="number" placeholder="1000" {...register('delay_message')} />
                <Input id="debounce_time" label="Debounce (seg)" type="number" placeholder="0" {...register('debounce_time')} />
              </div>
              <Input id="keyword_finish" label="Palavra p/ encerrar" placeholder="#sair" {...register('keyword_finish')} />
              <Input id="unknown_message" label="Mensagem desconhecida" placeholder="Não entendi..." {...register('unknown_message')} />
            </div>
          </section>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Opções</h3>
            <div className="space-y-3 rounded-lg bg-muted p-4">
              <Switch checked={enabled} onChange={setEnabled} label="Habilitado" />
              <Switch checked={listeningFromMe} onChange={setListeningFromMe} label="Escutar mensagens enviadas por mim" />
              <Switch checked={stopBotFromMe} onChange={setStopBotFromMe} label="Pausar bot quando eu enviar mensagem" />
              <Switch checked={keepOpen} onChange={setKeepOpen} label="Manter sessão aberta após finalizar" />
            </div>
          </section>
        </form>
      </SlideOver>

      {/* Settings SlideOver */}
      <SlideOver
        open={settingsModalOpen}
        onClose={() => setSettingsModalOpen(false)}
        title="Settings Globais"
        description={`Valores padrão para ${settingsInstance?.instance_name}`}
        size="md"
        footer={
          <div className="flex justify-between">
            <Button type="button" variant="outline" onClick={() => setSettingsModalOpen(false)}>Cancelar</Button>
            <Button onClick={handleSubmitSettings(onSubmitSettings)} loading={settingsSubmitting}>Salvar</Button>
          </div>
        }
      >
        <form onSubmit={handleSubmitSettings(onSubmitSettings)} className="space-y-6">
          <p className="text-xs text-muted-foreground">Estes valores serão usados como padrão quando o bot individual não definir um valor específico.</p>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Tempos</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <Input id="s_expire" label="Expirar (min)" type="number" placeholder="0" {...registerSettings('expire')} />
                <Input id="s_delay_message" label="Delay (ms)" type="number" placeholder="1000" {...registerSettings('delay_message')} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <Input id="s_debounce_time" label="Debounce (seg)" type="number" placeholder="0" {...registerSettings('debounce_time')} />
                <Input id="s_keyword_finish" label="Palavra p/ encerrar" placeholder="#sair" {...registerSettings('keyword_finish')} />
              </div>
              <Input id="s_unknown_message" label="Mensagem desconhecida" placeholder="Não entendi..." {...registerSettings('unknown_message')} />
            </div>
          </section>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Opções Padrão</h3>
            <div className="space-y-3 rounded-lg bg-muted p-4">
              <Switch checked={settingsListeningFromMe} onChange={setSettingsListeningFromMe} label="Escutar mensagens enviadas por mim" />
              <Switch checked={settingsStopBotFromMe} onChange={setSettingsStopBotFromMe} label="Pausar bot quando eu enviar mensagem" />
              <Switch checked={settingsKeepOpen} onChange={setSettingsKeepOpen} label="Manter sessão aberta após finalizar" />
            </div>
          </section>
        </form>
      </SlideOver>

      {/* Sessions SlideOver */}
      <SlideOver
        open={sessionsOpen}
        onClose={() => setSessionsOpen(false)}
        title="Sessões Typebot"
        description={sessionsInstance?.instance_name}
        size="lg"
      >
        <div>
          {sessions.length === 0 ? (
            <div className="text-center py-12">
              <div className="mx-auto w-12 h-12 rounded-md bg-muted flex items-center justify-center mb-3">
                <Users2 className="h-6 w-6 text-muted-foreground" />
              </div>
              <p className="text-muted-foreground text-sm">Nenhuma sessão ativa</p>
            </div>
          ) : (
            <div className="space-y-2">
              {sessions.map((session) => (
                <div key={session.id} className="flex items-center justify-between rounded-lg bg-muted p-3.5">
                  <div>
                    <p className="text-sm font-medium">{session.push_name || session.remote_jid}</p>
                    <p className="text-xs text-muted-foreground font-mono">{session.remote_jid}</p>
                    {session.created_at && <p className="text-[10px] text-muted-foreground mt-0.5">Criada: {new Date(session.created_at).toLocaleString('pt-BR')}</p>}
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant={session.status === 'opened' ? 'success' : session.status === 'paused' ? 'warning' : 'secondary'}>
                      {session.status === 'opened' ? 'Aberta' : session.status === 'paused' ? 'Pausada' : 'Fechada'}
                    </Badge>
                    {session.status === 'opened' && (
                      <>
                        <Button variant="ghost" size="sm" onClick={() => handleChangeStatus(session.remote_jid, 'paused')} title="Pausar">
                          <Pause className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => handleChangeStatus(session.remote_jid, 'closed')} title="Encerrar">
                          <Square className="h-3.5 w-3.5" />
                        </Button>
                      </>
                    )}
                    {session.status === 'paused' && (
                      <>
                        <Button variant="ghost" size="sm" onClick={() => handleChangeStatus(session.remote_jid, 'opened')} title="Retomar">
                          <Play className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => handleChangeStatus(session.remote_jid, 'closed')} title="Encerrar">
                          <Square className="h-3.5 w-3.5" />
                        </Button>
                      </>
                    )}
                    {session.status === 'closed' && (
                      <Button variant="ghost" size="sm" onClick={() => handleChangeStatus(session.remote_jid, 'opened')} title="Reabrir">
                        <Play className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </SlideOver>
    </div>
  )
}
