import { useEffect, useState, useRef, useCallback } from 'react'
import { toast } from 'sonner'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Plus, Trash2, QrCode, PowerOff, Send,
  RefreshCw, LogOut, Settings2, Phone, Search, Smartphone
} from 'lucide-react'
import { Button, Input, Card, CardContent, CardHeader, CardTitle, Modal, Badge, Select, Switch, SlideOver } from '@/components/ui'
import { instancesApi, serversApi } from '@/services/api'
import type { Instance, EvoServer, AdvancedSettings } from '@/types'

const instanceSchema = z.object({
  server_id: z.string().min(1, 'Selecione um servidor'),
  instance_name: z.string().min(1, 'Nome é obrigatório').regex(/^[a-zA-Z0-9_-]+$/, 'Apenas letras, números, _ e -'),
})

const messageSchema = z.object({
  number: z.string().min(10, 'Número inválido'),
  text: z.string().min(1, 'Mensagem é obrigatória'),
})

const pairSchema = z.object({
  phone: z.string().min(10, 'Número inválido').regex(/^\d+$/, 'Apenas números'),
})

type InstanceForm = z.infer<typeof instanceSchema>
type MessageForm = z.infer<typeof messageSchema>
type PairForm = z.infer<typeof pairSchema>

export default function InstancesPage() {
  const [instances, setInstances] = useState<Instance[]>([])
  const [servers, setServers] = useState<EvoServer[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [qrOpen, setQrOpen] = useState(false)
  const [sendOpen, setSendOpen] = useState(false)
  const [pairOpen, setPairOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [selectedInstance, setSelectedInstance] = useState<Instance | null>(null)
  const [qrCode, setQrCode] = useState<string | null>(null)
  const [pairCode, setPairCode] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  // Advanced settings state
  const [advSettings, setAdvSettings] = useState<AdvancedSettings>({
    alwaysOnline: false,
    rejectCall: false,
    msgRejectCall: '',
    readMessages: false,
    ignoreGroups: false,
    ignoreStatus: false,
  })
  const [savingSettings, setSavingSettings] = useState(false)

  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const { register, handleSubmit, reset, formState: { errors } } = useForm<InstanceForm>({
    resolver: zodResolver(instanceSchema),
  })

  const msgForm = useForm<MessageForm>({ resolver: zodResolver(messageSchema) })
  const pairForm = useForm<PairForm>({ resolver: zodResolver(pairSchema) })

  const loadData = useCallback(async () => {
    try {
      const [insts, srvs] = await Promise.all([instancesApi.list(), serversApi.list()])
      setInstances(insts)
      setServers(srvs)
    } catch {
      // silent on polling
    } finally {
      setLoading(false)
    }
  }, [])

  // Initial load + polling every 5s
  useEffect(() => {
    loadData()
    pollRef.current = setInterval(loadData, 5000)
    return () => {
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [loadData])

  const onCreateSubmit = async (data: InstanceForm) => {
    setSubmitting(true)
    try {
      await instancesApi.create(data)
      toast.success('Instância criada!')
      setCreateOpen(false)
      reset()
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao criar instância')
    } finally {
      setSubmitting(false)
    }
  }

  const handleConnect = async (inst: Instance) => {
    try {
      await instancesApi.connect(inst.id)
      toast.success('Conectando...')
      const qr = await instancesApi.qrcode(inst.id)
      setSelectedInstance(inst)
      setQrCode(qr.qrcode)
      setQrOpen(true)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao conectar')
    }
  }

  const handleDisconnect = async (id: string) => {
    try {
      await instancesApi.disconnect(id)
      toast.success('Desconectado!')
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao desconectar')
    }
  }

  const handleLogout = async (id: string) => {
    if (!confirm('Fazer logout irá limpar a sessão do WhatsApp. Continuar?')) return
    try {
      await instancesApi.logout(id)
      toast.success('Logout realizado!')
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao fazer logout')
    }
  }

  const handleReconnect = async (id: string) => {
    try {
      await instancesApi.reconnect(id)
      toast.success('Reconectando...')
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao reconectar')
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Tem certeza que deseja excluir esta instância?')) return
    try {
      await instancesApi.delete(id)
      toast.success('Instância excluída!')
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao excluir')
    }
  }

  const openSendMessage = (inst: Instance) => {
    setSelectedInstance(inst)
    msgForm.reset()
    setSendOpen(true)
  }

  const onSendMessage = async (data: MessageForm) => {
    if (!selectedInstance) return
    setSubmitting(true)
    try {
      await instancesApi.sendMessage(selectedInstance.id, data)
      toast.success('Mensagem enviada!')
      setSendOpen(false)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao enviar')
    } finally {
      setSubmitting(false)
    }
  }

  const refreshQR = async () => {
    if (!selectedInstance) return
    try {
      const qr = await instancesApi.qrcode(selectedInstance.id)
      setQrCode(qr.qrcode)
    } catch {
      toast.error('Erro ao atualizar QR Code')
    }
  }

  // Pairing code
  const openPairModal = (inst: Instance) => {
    setSelectedInstance(inst)
    setPairCode(null)
    pairForm.reset()
    setPairOpen(true)
  }

  const onPairSubmit = async (data: PairForm) => {
    if (!selectedInstance) return
    setSubmitting(true)
    try {
      // First connect
      await instancesApi.connect(selectedInstance.id)
      // Then pair
      const resp = await instancesApi.pair(selectedInstance.id, { phone: data.phone })
      const code = resp?.data?.PairingCode || resp?.data?.pairingCode || resp?.data
      setPairCode(typeof code === 'string' ? code : JSON.stringify(code))
      toast.success('Código de pareamento gerado!')
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao gerar código')
    } finally {
      setSubmitting(false)
    }
  }

  // Advanced settings
  const openSettings = async (inst: Instance) => {
    setSelectedInstance(inst)
    try {
      const resp: any = await instancesApi.getAdvancedSettings(inst.id)
      const data = resp?.data || resp
      setAdvSettings({
        alwaysOnline: data?.alwaysOnline ?? false,
        rejectCall: data?.rejectCall ?? false,
        msgRejectCall: data?.msgRejectCall ?? '',
        readMessages: data?.readMessages ?? false,
        ignoreGroups: data?.ignoreGroups ?? false,
        ignoreStatus: data?.ignoreStatus ?? false,
      })
      setSettingsOpen(true)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao carregar configurações')
    }
  }

  const saveSettings = async () => {
    if (!selectedInstance) return
    setSavingSettings(true)
    try {
      await instancesApi.updateAdvancedSettings(selectedInstance.id, advSettings)
      toast.success('Configurações salvas!')
      setSettingsOpen(false)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao salvar')
    } finally {
      setSavingSettings(false)
    }
  }

  // Filter instances
  const filtered = instances.filter((inst) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      inst.instance_name?.toLowerCase().includes(q) ||
      inst.server_name?.toLowerCase().includes(q) ||
      inst.push_name?.toLowerCase().includes(q) ||
      inst.phone?.toLowerCase().includes(q)
    )
  })

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Instâncias WhatsApp</h1>
          <p className="text-muted-foreground mt-1">Gerencie suas conexões WhatsApp</p>
        </div>
        <Button onClick={() => { reset(); setCreateOpen(true) }} className="shrink-0">
          <Plus className="h-4 w-4" /> Nova Instância
        </Button>
      </div>

      {/* Search */}
      {instances.length > 0 && (
        <div className="relative max-w-sm">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Buscar instâncias..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-input bg-background pl-11 pr-4 py-2.5 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary transition-colors"
          />
        </div>
      )}

      {instances.length === 0 ? (
        <Card className="border-dashed border-2 shadow-none">
          <CardContent className="py-16 text-center">
            <div className="mx-auto w-12 h-12 rounded-md bg-muted flex items-center justify-center mb-4">
              <Smartphone className="h-6 w-6 text-muted-foreground" />
            </div>
            <h3 className="text-base font-medium mb-1.5">Nenhuma instância</h3>
            <p className="text-muted-foreground text-sm mb-8">Crie sua primeira instância WhatsApp para começar</p>
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" /> Criar Instância
            </Button>
          </CardContent>
        </Card>
      ) : filtered.length === 0 ? (
        <Card className="">
          <CardContent className="py-10 text-center">
            <p className="text-muted-foreground font-medium">Nenhuma instância encontrada para "{search}"</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {filtered.map((inst) => (
            <Card key={inst.id} className="group transition-all duration-200 hover-lift">
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="h-9 w-9 rounded-md bg-muted flex items-center justify-center">
                      <Smartphone className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <CardTitle className="text-sm font-medium">{inst.instance_name}</CardTitle>
                      <p className="text-xs text-muted-foreground mt-0.5">{inst.server_name}</p>
                    </div>
                  </div>
                  <Badge variant={inst.connection_status === 'connected' ? 'success' : inst.connection_status === 'connecting' ? 'warning' : 'secondary'}>
                    {inst.connection_status === 'connected' ? 'Online' : inst.connection_status === 'connecting' ? 'Conectando' : 'Offline'}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="pt-0">
                {(inst.phone || inst.push_name) && (
                  <div className="text-xs text-muted-foreground mb-3 rounded-lg bg-muted px-3 py-2.5">
                    {inst.push_name && <p className="font-semibold text-foreground/80">{inst.push_name}</p>}
                    {inst.phone && <p className="font-mono text-muted-foreground">{inst.phone}</p>}
                  </div>
                )}

                {/* Integration badges */}
                <div className="flex gap-1.5 mb-3">
                  {inst.webhook_configured && <Badge variant="outline" className="text-[10px]">Webhook</Badge>}
                  {inst.has_chatwoot && <Badge variant="outline" className="text-[10px]">Chatwoot</Badge>}
                  {inst.has_typebot && <Badge variant="outline" className="text-[10px]">Typebot</Badge>}
                </div>

                {/* Actions */}
                <div className="flex items-center gap-1 pt-3 border-t border-border">
                  {inst.connection_status !== 'connected' ? (
                    <>
                      <Button variant="ghost" size="sm" onClick={() => handleConnect(inst)} className="text-xs gap-1.5">
                        <QrCode className="h-3.5 w-3.5" /> QR Code
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => openPairModal(inst)} className="text-xs gap-1.5">
                        <Phone className="h-3.5 w-3.5" /> Parear
                      </Button>
                    </>
                  ) : (
                    <>
                      <Button variant="ghost" size="sm" onClick={() => openSendMessage(inst)} className="text-xs gap-1.5">
                        <Send className="h-3.5 w-3.5" /> Enviar
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => handleReconnect(inst.id)} title="Reconectar">
                        <RefreshCw className="h-3.5 w-3.5" />
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => handleDisconnect(inst.id)} title="Desconectar">
                        <PowerOff className="h-3.5 w-3.5" />
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => handleLogout(inst.id)} title="Logout">
                        <LogOut className="h-3.5 w-3.5" />
                      </Button>
                    </>
                  )}
                  <div className="flex-1" />
                  <Button variant="ghost" size="sm" onClick={() => openSettings(inst)} title="Configurações">
                    <Settings2 className="h-3.5 w-3.5" />
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => handleDelete(inst.id)} className="text-destructive hover:text-destructive" title="Excluir">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Create Instance SlideOver */}
      <SlideOver
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Nova Instância"
        description="Crie uma nova instância WhatsApp"
        size="sm"
        footer={
          <div className="flex justify-between">
            <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>Cancelar</Button>
            <Button onClick={handleSubmit(onCreateSubmit)} loading={submitting}>Criar Instância</Button>
          </div>
        }
      >
        <form onSubmit={handleSubmit(onCreateSubmit)} className="space-y-4">
          <Select
            id="server_id"
            label="Servidor"
            placeholder="Selecione um servidor"
            options={servers.map(s => ({ value: s.id, label: s.name }))}
            error={errors.server_id?.message}
            {...register('server_id')}
          />
          <Input
            id="instance_name"
            label="Nome da Instância"
            placeholder="minha-instancia"
            error={errors.instance_name?.message}
            {...register('instance_name')}
          />
        </form>
      </SlideOver>

      {/* QR Code Modal */}
      <Modal open={qrOpen} onClose={() => { setQrOpen(false); loadData() }} title="QR Code" description={`Escaneie com o WhatsApp - ${selectedInstance?.instance_name}`}>
        <div className="flex flex-col items-center gap-6">
          {qrCode ? (
            <div className="rounded-xl bg-white p-5 shadow-inner">
              <img src={qrCode} alt="QR Code" className="w-56 h-56" />
            </div>
          ) : (
            <div className="w-64 h-64 flex items-center justify-center bg-muted rounded-xl">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
            </div>
          )}
          <div className="text-center space-y-3">
            <p className="text-xs text-muted-foreground">Abra o WhatsApp no celular e escaneie o QR Code</p>
            <Button variant="outline" size="sm" onClick={refreshQR}>
              <RefreshCw className="h-3.5 w-3.5" /> Atualizar QR Code
            </Button>
          </div>
        </div>
      </Modal>

      {/* Pair Code Modal */}
      <Modal open={pairOpen} onClose={() => { setPairOpen(false); loadData() }} title="Parear por Telefone" description={`Conectar ${selectedInstance?.instance_name}`}>
        {pairCode ? (
          <div className="flex flex-col items-center gap-6 py-4">
            <div className="rounded-lg bg-muted px-8 py-6 border border-border">
              <p className="text-3xl font-mono font-semibold tracking-[0.3em] select-all text-foreground">{pairCode}</p>
            </div>
            <div className="text-center space-y-1.5">
              <p className="text-sm font-semibold">Digite este código no WhatsApp</p>
              <p className="text-xs text-muted-foreground">Dispositivos conectados &rarr; Conectar dispositivo &rarr; Conectar com número de telefone</p>
            </div>
          </div>
        ) : (
          <form onSubmit={pairForm.handleSubmit(onPairSubmit)} className="space-y-5">
            <Input
              id="phone"
              label="Número de Telefone"
              placeholder="5511999998888"
              error={pairForm.formState.errors.phone?.message}
              {...pairForm.register('phone')}
            />
            <p className="text-xs text-muted-foreground">Número completo com código do país, sem + ou espaços</p>
            <div className="flex justify-end gap-3 pt-5 border-t border-border">
              <Button type="button" variant="outline" onClick={() => setPairOpen(false)}>Cancelar</Button>
              <Button type="submit" loading={submitting}>Gerar Código</Button>
            </div>
          </form>
        )}
      </Modal>

      {/* Send Message SlideOver */}
      <SlideOver
        open={sendOpen}
        onClose={() => setSendOpen(false)}
        title="Enviar Mensagem"
        description={`Via ${selectedInstance?.instance_name}`}
        size="md"
        footer={
          <div className="flex justify-between">
            <Button type="button" variant="outline" onClick={() => setSendOpen(false)}>Cancelar</Button>
            <Button onClick={msgForm.handleSubmit(onSendMessage)} loading={submitting}>
              <Send className="h-3.5 w-3.5" /> Enviar
            </Button>
          </div>
        }
      >
        <form onSubmit={msgForm.handleSubmit(onSendMessage)} className="space-y-4">
          <Input
            id="number"
            label="Número"
            placeholder="5511999998888"
            error={msgForm.formState.errors.number?.message}
            {...msgForm.register('number')}
          />
          <div className="space-y-2">
            <label htmlFor="text" className="text-sm font-medium text-foreground">Mensagem</label>
            <textarea
              id="text"
              className="flex w-full rounded-lg border border-input bg-background px-3 py-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary min-h-[100px] resize-y transition-colors"
              placeholder="Digite sua mensagem..."
              {...msgForm.register('text')}
            />
            {msgForm.formState.errors.text && <p className="text-xs font-medium text-destructive">{msgForm.formState.errors.text.message}</p>}
          </div>
        </form>
      </SlideOver>

      {/* Advanced Settings SlideOver */}
      <SlideOver
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        title="Configurações Avançadas"
        description={selectedInstance?.instance_name}
        size="md"
        footer={
          <div className="flex justify-between">
            <Button variant="outline" onClick={() => setSettingsOpen(false)}>Cancelar</Button>
            <Button onClick={saveSettings} loading={savingSettings}>Salvar</Button>
          </div>
        }
      >
        <div className="space-y-6">
          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Comportamento</h3>
            <div className="space-y-3 rounded-lg bg-muted p-4">
              <Switch
                checked={advSettings.alwaysOnline}
                onChange={(v) => setAdvSettings(prev => ({ ...prev, alwaysOnline: v }))}
                label="Sempre online"
              />
              <Switch
                checked={advSettings.readMessages}
                onChange={(v) => setAdvSettings(prev => ({ ...prev, readMessages: v }))}
                label="Marcar mensagens como lidas"
              />
              <Switch
                checked={advSettings.ignoreGroups}
                onChange={(v) => setAdvSettings(prev => ({ ...prev, ignoreGroups: v }))}
                label="Ignorar mensagens de grupos"
              />
              <Switch
                checked={advSettings.ignoreStatus}
                onChange={(v) => setAdvSettings(prev => ({ ...prev, ignoreStatus: v }))}
                label="Ignorar status/stories"
              />
            </div>
          </section>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Chamadas</h3>
            <div className="space-y-3 rounded-lg bg-muted p-4">
              <Switch
                checked={advSettings.rejectCall}
                onChange={(v) => setAdvSettings(prev => ({ ...prev, rejectCall: v }))}
                label="Rejeitar chamadas"
              />
            </div>
            {advSettings.rejectCall && (
              <div className="mt-3">
                <Input
                  id="msgRejectCall"
                  label="Mensagem ao rejeitar chamada"
                  placeholder="Não posso atender agora..."
                  value={advSettings.msgRejectCall}
                  onChange={(e) => setAdvSettings(prev => ({ ...prev, msgRejectCall: e.target.value }))}
                />
              </div>
            )}
          </section>
        </div>
      </SlideOver>
    </div>
  )
}
