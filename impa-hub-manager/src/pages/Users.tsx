import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Users, Plus, Trash2, Key, Settings2, ShieldCheck, Shield, User as UserIcon, Search } from 'lucide-react'
import { Button, Input, Card, CardContent, CardHeader, CardTitle, Badge, Select, Switch, SkeletonTable, EmptyState, ConfirmDialog, Pagination, SlideOver } from '@/components/ui'
import { adminApi } from '@/services/api'
import type { User, UserRole } from '@/types'

const createUserSchema = z.object({
  name: z.string().min(1, 'Nome é obrigatório'),
  email: z.string().email('Email inválido'),
  password: z.string().min(6, 'Mínimo 6 caracteres'),
  role: z.enum(['superadmin', 'admin', 'user']),
  max_instances: z.coerce.number().min(0),
  max_chatwoot_conns: z.coerce.number().min(0),
  max_evo_servers: z.coerce.number().min(0),
})

type CreateUserForm = z.infer<typeof createUserSchema>

const quotasSchema = z.object({
  max_instances: z.coerce.number().min(0),
  max_chatwoot_conns: z.coerce.number().min(0),
  max_evo_servers: z.coerce.number().min(0),
})

type QuotasForm = z.infer<typeof quotasSchema>

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [quotasOpen, setQuotasOpen] = useState(false)
  const [resetPwOpen, setResetPwOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [newPassword, setNewPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [canUseChatwoot, setCanUseChatwoot] = useState(true)
  const [canUseTypebot, setCanUseTypebot] = useState(true)
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [userToDelete, setUserToDelete] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const itemsPerPage = 10

  const createForm = useForm<CreateUserForm>({
    resolver: zodResolver(createUserSchema),
    defaultValues: { role: 'user', max_instances: 5, max_chatwoot_conns: 5, max_evo_servers: 2 },
  })

  const quotasForm = useForm<QuotasForm>({ resolver: zodResolver(quotasSchema) })

  const loadUsers = async () => {
    try {
      const data = await adminApi.listUsers()
      setUsers(data)
    } catch {
      toast.error('Erro ao carregar usuários')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadUsers() }, [])

  const onCreateUser = async (data: CreateUserForm) => {
    setSubmitting(true)
    try {
      await adminApi.createUser({
        ...data,
        can_use_chatwoot: canUseChatwoot,
        can_use_typebot: canUseTypebot,
      })
      toast.success('Usuário criado!')
      setCreateOpen(false)
      createForm.reset()
      loadUsers()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao criar usuário')
    } finally {
      setSubmitting(false)
    }
  }

  const openQuotas = (user: User) => {
    setSelectedUser(user)
    quotasForm.reset({
      max_instances: user.max_instances,
      max_chatwoot_conns: user.max_chatwoot_conns,
      max_evo_servers: user.max_evo_servers,
    })
    setCanUseChatwoot(user.can_use_chatwoot)
    setCanUseTypebot(user.can_use_typebot)
    setQuotasOpen(true)
  }

  const onUpdateQuotas = async (data: QuotasForm) => {
    if (!selectedUser) return
    setSubmitting(true)
    try {
      await adminApi.updateQuotas(selectedUser.id, {
        ...data,
        can_use_chatwoot: canUseChatwoot,
        can_use_typebot: canUseTypebot,
      })
      toast.success('Cotas atualizadas!')
      setQuotasOpen(false)
      loadUsers()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao atualizar cotas')
    } finally {
      setSubmitting(false)
    }
  }

  const handleResetPassword = async () => {
    if (!selectedUser || !newPassword) return
    setSubmitting(true)
    try {
      await adminApi.resetPassword(selectedUser.id, newPassword)
      toast.success('Senha resetada!')
      setResetPwOpen(false)
      setNewPassword('')
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao resetar senha')
    } finally {
      setSubmitting(false)
    }
  }

  const handleToggleActive = async (user: User) => {
    try {
      await adminApi.toggleActive(user.id, !user.is_active)
      toast.success(user.is_active ? 'Usuário desativado' : 'Usuário ativado')
      loadUsers()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao alterar status')
    }
  }

  const handleDelete = async (userId: string) => {
    setUserToDelete(userId)
    setDeleteConfirmOpen(true)
  }

  const confirmDelete = async () => {
    if (!userToDelete) return
    try {
      await adminApi.deleteUser(userToDelete)
      toast.success('Usuário excluído!')
      loadUsers()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao excluir')
    } finally {
      setDeleteConfirmOpen(false)
      setUserToDelete(null)
    }
  }

  const roleIcon = (role: UserRole) => {
    switch (role) {
      case 'superadmin': return <ShieldCheck className="h-4 w-4 text-primary" />
      case 'admin': return <Shield className="h-4 w-4 text-amber-500" />
      default: return <UserIcon className="h-4 w-4 text-muted-foreground" />
    }
  }

  const roleLabel: Record<UserRole, string> = {
    superadmin: 'Super Admin',
    admin: 'Admin',
    user: 'Usuário',
  }

  const filteredUsers = useMemo(() => {
    if (!search.trim()) return users
    const q = search.toLowerCase()
    return users.filter(u =>
      u.name.toLowerCase().includes(q) ||
      u.email.toLowerCase().includes(q) ||
      u.role.toLowerCase().includes(q)
    )
  }, [users, search])

  const totalPages = Math.max(1, Math.ceil(filteredUsers.length / itemsPerPage))
  const paginatedUsers = filteredUsers.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage)

  // Reset page when search changes
  useEffect(() => { setCurrentPage(1) }, [search])

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Gerenciar Usuários</h1>
            <p className="text-muted-foreground mt-1">Administre usuários e suas cotas</p>
          </div>
        </div>
        <Card className=" overflow-hidden">
          <SkeletonTable rows={5} cols={6} />
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Gerenciar Usuários</h1>
          <p className="text-muted-foreground mt-1">Administre usuários e suas cotas</p>
        </div>
        <Button onClick={() => { createForm.reset(); setCanUseChatwoot(true); setCanUseTypebot(true); setCreateOpen(true) }} className="shrink-0">
          <Plus className="h-4 w-4" /> Novo Usuário
        </Button>
      </div>

      {users.length === 0 ? (
        <EmptyState
          icon={Users}
          title="Nenhum usuário"
          description="Crie o primeiro usuário do sistema"
          actionLabel="Criar Usuário"
          onAction={() => setCreateOpen(true)}
        />
      ) : (
        <>
          {/* Search */}
          <div className="relative max-w-xs">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Buscar usuário..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="h-9 w-full rounded-lg border border-input bg-background pl-9 pr-3 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1 focus:ring-offset-background"
            />
          </div>

          <Card className=" overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/30">
                  <th className="px-4 py-3.5 text-left text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Usuário</th>
                  <th className="px-4 py-3.5 text-left text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Perfil</th>
                  <th className="px-4 py-3.5 text-left text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Status</th>
                  <th className="px-4 py-3.5 text-left text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Cotas</th>
                  <th className="px-4 py-3.5 text-left text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Integrações</th>
                  <th className="px-4 py-3.5 text-right text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Ações</th>
                </tr>
              </thead>
              <tbody>
                {paginatedUsers.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="text-center py-12 text-muted-foreground text-sm">
                      Nenhum usuário encontrado para "{search}"
                    </td>
                  </tr>
                ) : paginatedUsers.map((user) => (
                  <tr key={user.id} className="border-b border-border last:border-0 hover:bg-muted/20 transition-colors">
                    <td className="px-4 py-3.5">
                      <div className="flex items-center gap-3">
                        <div className="h-8 w-8 rounded-md bg-muted flex items-center justify-center">
                          <span className="text-xs font-medium text-foreground">{user.name.charAt(0).toUpperCase()}</span>
                        </div>
                        <div>
                          <p className="font-semibold">{user.name}</p>
                          <p className="text-xs text-muted-foreground">{user.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1.5">
                        {roleIcon(user.role)}
                        <span className="text-xs">{roleLabel[user.role]}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant={user.is_active ? 'success' : 'destructive'} className="cursor-pointer" onClick={() => handleToggleActive(user)}>
                        {user.is_active ? 'Ativo' : 'Inativo'}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">
                      <div className="text-xs space-y-0.5 text-muted-foreground">
                        <p>Inst: <span className="text-foreground font-medium">{user.max_instances}</span> | Srv: <span className="text-foreground font-medium">{user.max_evo_servers}</span></p>
                        <p>Chatwoot: <span className="text-foreground font-medium">{user.max_chatwoot_conns}</span></p>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-1">
                        <Badge variant={user.can_use_chatwoot ? 'success' : 'secondary'} className="text-[10px]">CW</Badge>
                        <Badge variant={user.can_use_typebot ? 'success' : 'secondary'} className="text-[10px]">TB</Badge>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="sm" onClick={() => openQuotas(user)} title="Cotas">
                          <Settings2 className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => { setSelectedUser(user); setNewPassword(''); setResetPwOpen(true) }} title="Resetar senha">
                          <Key className="h-3.5 w-3.5" />
                        </Button>
                        {user.role !== 'superadmin' && (
                          <Button variant="ghost" size="sm" onClick={() => handleDelete(user.id)} className="text-destructive hover:text-destructive" title="Excluir">
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {filteredUsers.length > itemsPerPage && (
            <div className="px-4 pb-4">
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPageChange={setCurrentPage}
                totalItems={filteredUsers.length}
                itemsPerPage={itemsPerPage}
              />
            </div>
          )}
        </Card>
        </>
      )}

      {/* Create User SlideOver */}
      <SlideOver
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Novo Usuário"
        description="Crie um novo usuário do sistema"
        size="lg"
        footer={
          <div className="flex justify-between">
            <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>Cancelar</Button>
            <Button onClick={createForm.handleSubmit(onCreateUser)} loading={submitting}>Criar Usuário</Button>
          </div>
        }
      >
        <form onSubmit={createForm.handleSubmit(onCreateUser)} className="space-y-6">
          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Dados Pessoais</h3>
            <div className="space-y-3">
              <Input id="name" label="Nome" error={createForm.formState.errors.name?.message} {...createForm.register('name')} />
              <Input id="email" label="Email" type="email" error={createForm.formState.errors.email?.message} {...createForm.register('email')} />
              <div className="grid grid-cols-2 gap-3">
                <Input id="password" label="Senha" type="password" error={createForm.formState.errors.password?.message} {...createForm.register('password')} />
                <Select
                  id="role"
                  label="Perfil"
                  options={[
                    { value: 'user', label: 'Usuário' },
                    { value: 'admin', label: 'Admin' },
                    { value: 'superadmin', label: 'Super Admin' },
                  ]}
                  {...createForm.register('role')}
                />
              </div>
            </div>
          </section>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Limites</h3>
            <div className="space-y-3">
              <Input id="max_instances" label="Máx. Instâncias" type="number" {...createForm.register('max_instances')} />
              <div className="grid grid-cols-2 gap-3">
                <Input id="max_evo_servers" label="Máx. Servidores" type="number" {...createForm.register('max_evo_servers')} />
                <Input id="max_chatwoot_conns" label="Máx. Chatwoot" type="number" {...createForm.register('max_chatwoot_conns')} />
              </div>
            </div>
          </section>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Integrações</h3>
            <div className="space-y-3 rounded-lg bg-muted p-4">
              <Switch checked={canUseChatwoot} onChange={setCanUseChatwoot} label="Chatwoot" />
              <Switch checked={canUseTypebot} onChange={setCanUseTypebot} label="Typebot" />
            </div>
          </section>
        </form>
      </SlideOver>

      {/* Quotas SlideOver */}
      <SlideOver
        open={quotasOpen}
        onClose={() => setQuotasOpen(false)}
        title="Editar Cotas & Permissões"
        description={selectedUser?.name}
        size="md"
        footer={
          <div className="flex justify-between">
            <Button type="button" variant="outline" onClick={() => setQuotasOpen(false)}>Cancelar</Button>
            <Button onClick={quotasForm.handleSubmit(onUpdateQuotas)} loading={submitting}>Salvar</Button>
          </div>
        }
      >
        <form onSubmit={quotasForm.handleSubmit(onUpdateQuotas)} className="space-y-6">
          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Limites</h3>
            <div className="space-y-3">
              <Input id="q_max_instances" label="Máx. Instâncias" type="number" {...quotasForm.register('max_instances')} />
              <div className="grid grid-cols-2 gap-3">
                <Input id="q_max_evo_servers" label="Máx. Servidores" type="number" {...quotasForm.register('max_evo_servers')} />
                <Input id="q_max_chatwoot_conns" label="Máx. Chatwoot" type="number" {...quotasForm.register('max_chatwoot_conns')} />
              </div>
            </div>
          </section>

          <section>
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">Integrações</h3>
            <div className="space-y-3 rounded-lg bg-muted p-4">
              <Switch checked={canUseChatwoot} onChange={setCanUseChatwoot} label="Chatwoot" />
              <Switch checked={canUseTypebot} onChange={setCanUseTypebot} label="Typebot" />
            </div>
          </section>
        </form>
      </SlideOver>

      {/* Reset Password SlideOver */}
      <SlideOver
        open={resetPwOpen}
        onClose={() => setResetPwOpen(false)}
        title="Resetar Senha"
        description={selectedUser?.name}
        size="sm"
        footer={
          <div className="flex justify-between">
            <Button variant="outline" onClick={() => setResetPwOpen(false)}>Cancelar</Button>
            <Button onClick={handleResetPassword} loading={submitting} disabled={newPassword.length < 6}>Resetar Senha</Button>
          </div>
        }
      >
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">Defina uma nova senha para o usuário. A senha anterior será substituída imediatamente.</p>
          <Input
            id="new_password"
            label="Nova Senha"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            placeholder="Mínimo 6 caracteres"
          />
        </div>
      </SlideOver>

      {/* Delete Confirm Dialog */}
      <ConfirmDialog
        open={deleteConfirmOpen}
        onClose={() => { setDeleteConfirmOpen(false); setUserToDelete(null) }}
        onConfirm={confirmDelete}
        title="Excluir Usuário"
        description="Tem certeza que deseja excluir este usuário? Esta ação não pode ser desfeita."
        confirmLabel="Excluir"
        variant="danger"
      />
    </div>
  )
}
