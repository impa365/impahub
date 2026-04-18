import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Trash2, Plug, Pencil, Server } from 'lucide-react'
import { Button, Input, Card, CardContent, CardHeader, CardTitle, SlideOver, Badge, ConfirmDialog } from '@/components/ui'
import { serversApi } from '@/services/api'
import type { EvoServer } from '@/types'

const serverSchema = z.object({
  name: z.string().min(1, 'Nome é obrigatório'),
  base_url: z.string().url('URL inválida'),
  global_api_key: z.string().min(1, 'API Key é obrigatória'),
})

type ServerForm = z.infer<typeof serverSchema>

export default function ServersPage() {
  const [servers, setServers] = useState<EvoServer[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editServer, setEditServer] = useState<EvoServer | null>(null)
  const [testing, setTesting] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const { register, handleSubmit, reset, formState: { errors } } = useForm<ServerForm>({
    resolver: zodResolver(serverSchema),
  })

  const loadServers = async () => {
    try {
      const data = await serversApi.list()
      setServers(data)
    } catch {
      toast.error('Erro ao carregar servidores')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadServers() }, [])

  const openCreate = () => {
    setEditServer(null)
    reset({ name: '', base_url: '', global_api_key: '' })
    setModalOpen(true)
  }

  const openEdit = (server: EvoServer) => {
    setEditServer(server)
    reset({ name: server.name, base_url: server.base_url, global_api_key: '' })
    setModalOpen(true)
  }

  const onSubmit = async (data: ServerForm) => {
    setSubmitting(true)
    try {
      if (editServer) {
        await serversApi.update(editServer.id, data)
        toast.success('Servidor atualizado!')
      } else {
        await serversApi.create(data)
        toast.success('Servidor criado!')
      }
      setModalOpen(false)
      loadServers()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao salvar servidor')
    } finally {
      setSubmitting(false)
    }
  }

  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [serverToDelete, setServerToDelete] = useState<string | null>(null)

  const handleDelete = (id: string) => {
    setServerToDelete(id)
    setDeleteConfirmOpen(true)
  }

  const confirmDelete = async () => {
    if (!serverToDelete) return
    try {
      await serversApi.delete(serverToDelete)
      toast.success('Servidor excluído!')
      loadServers()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao excluir')
    } finally {
      setDeleteConfirmOpen(false)
      setServerToDelete(null)
    }
  }

  const handleTest = async (id: string) => {
    setTesting(id)
    try {
      const res = await serversApi.test(id)
      if (res.success) {
        toast.success('Conexão bem-sucedida!')
      } else {
        toast.error(res.message || 'Falha na conexão')
      }
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao testar conexão')
    } finally {
      setTesting(null)
    }
  }

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
          <h1 className="text-3xl font-bold tracking-tight">Servidores EVO</h1>
          <p className="text-muted-foreground mt-1">Gerencie seus servidores Evolution GO</p>
        </div>
        <Button onClick={openCreate} className="shrink-0">
          <Plus className="h-4 w-4" /> Novo Servidor
        </Button>
      </div>

      {servers.length === 0 ? (
        <Card className="border-dashed border-2 shadow-none">
          <CardContent className="py-16 text-center">
            <div className="mx-auto w-12 h-12 rounded-md bg-muted flex items-center justify-center mb-4">
              <Server className="h-6 w-6 text-muted-foreground" />
            </div>
            <h3 className="text-base font-medium mb-1.5">Nenhum servidor</h3>
            <p className="text-muted-foreground text-sm mb-8">Adicione seu primeiro servidor Evolution GO</p>
            <Button onClick={openCreate}>
              <Plus className="h-4 w-4" /> Adicionar Servidor
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {servers.map((server) => (
            <Card key={server.id} className="overflow-hidden hover-lift">
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="h-9 w-9 rounded-md bg-muted flex items-center justify-center">
                      <Server className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <CardTitle className="text-sm font-medium">{server.name}</CardTitle>
                      <p className="text-xs text-muted-foreground mt-0.5 break-all max-w-[200px] truncate">{server.base_url}</p>
                    </div>
                  </div>
                  <Badge variant={server.is_active ? 'success' : 'secondary'}>
                    {server.is_active ? 'Ativo' : 'Inativo'}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="pt-0">
                <div className="flex items-center gap-1 pt-3 border-t border-border">
                  <Button variant="ghost" size="sm" onClick={() => handleTest(server.id)} loading={testing === server.id} className="text-xs gap-1.5">
                    <Plug className="h-3.5 w-3.5" /> Testar
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => openEdit(server)} className="text-xs gap-1.5">
                    <Pencil className="h-3.5 w-3.5" /> Editar
                  </Button>
                  <div className="flex-1" />
                  <Button variant="ghost" size="sm" onClick={() => handleDelete(server.id)} className="text-destructive hover:text-destructive" title="Excluir">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Create/Edit SlideOver */}
      <SlideOver
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editServer ? 'Editar Servidor' : 'Novo Servidor'}
        description="Configure a conexão com um servidor Evolution GO"
        size="md"
        footer={
          <div className="flex justify-between">
            <Button type="button" variant="outline" onClick={() => setModalOpen(false)}>Cancelar</Button>
            <Button onClick={handleSubmit(onSubmit)} loading={submitting}>{editServer ? 'Salvar' : 'Criar Servidor'}</Button>
          </div>
        }
      >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <Input id="name" label="Nome" placeholder="Meu Servidor" error={errors.name?.message} {...register('name')} />
          <Input id="base_url" label="URL Base" placeholder="https://evo.example.com" error={errors.base_url?.message} {...register('base_url')} />
          <Input id="global_api_key" label="Global API Key" placeholder="sua-api-key" error={errors.global_api_key?.message} {...register('global_api_key')} />
          {editServer && <p className="text-xs text-muted-foreground">Deixe a API Key em branco para manter a atual</p>}
        </form>
      </SlideOver>

      {/* Delete Confirm Dialog */}
      <ConfirmDialog
        open={deleteConfirmOpen}
        onClose={() => { setDeleteConfirmOpen(false); setServerToDelete(null) }}
        onConfirm={confirmDelete}
        title="Excluir Servidor"
        description="Tem certeza que deseja excluir este servidor? Esta ação não pode ser desfeita."
        confirmLabel="Excluir"
        variant="danger"
      />
    </div>
  )
}
