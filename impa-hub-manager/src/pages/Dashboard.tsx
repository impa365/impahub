import { useEffect, useState } from 'react'
import { Server, Smartphone, Users, Activity, ArrowUpRight } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, Badge, Skeleton, SkeletonCard } from '@/components/ui'
import { useAuthStore } from '@/store/authStore'
import { serversApi, instancesApi } from '@/services/api'
import { adminApi } from '@/services/api'
import { useNavigate } from 'react-router-dom'
import type { EvoServer, Instance } from '@/types'

export default function DashboardPage() {
  const { user } = useAuthStore()
  const navigate = useNavigate()
  const isSuperAdmin = user?.role === 'superadmin'

  const [servers, setServers] = useState<EvoServer[]>([])
  const [instances, setInstances] = useState<Instance[]>([])
  const [usersCount, setUsersCount] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const load = async () => {
      try {
        const [srvs, insts] = await Promise.all([
          serversApi.list(),
          instancesApi.list(),
        ])
        setServers(srvs)
        setInstances(insts)

        if (isSuperAdmin) {
          const users = await adminApi.listUsers()
          setUsersCount(users.length)
        }
      } catch {
        // silently handle
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [isSuperAdmin])

  const connectedInstances = instances.filter((i) => i.connection_status === 'connected').length

  if (loading) {
    return (
      <div className="space-y-8">
        <div>
          <Skeleton className="h-8 w-40 mb-2" />
          <Skeleton className="h-4 w-56" />
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <SkeletonCard key={i} />
          ))}
        </div>
        <div className="grid gap-6 lg:grid-cols-5">
          <Card className="lg:col-span-3 ">
            <CardContent className="p-6 space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-12 rounded-lg" />
              ))}
            </CardContent>
          </Card>
          <Card className="lg:col-span-2 ">
            <CardContent className="p-6 space-y-4">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-2 rounded-full" />
              <Skeleton className="h-4 w-24 mt-4" />
              <Skeleton className="h-2 rounded-full" />
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  const statCards = [
    { title: 'Servidores', value: servers.length, sub: `${servers.filter(s => s.is_active).length} ativos`, icon: Server, iconBg: 'bg-blue-500/15', iconColor: 'text-blue-400' },
    { title: 'Instâncias', value: instances.length, sub: `${connectedInstances} conectadas`, icon: Smartphone, iconBg: 'bg-violet-500/15', iconColor: 'text-violet-400' },
    { title: 'Conectadas', value: connectedInstances, sub: 'WhatsApp online', icon: Activity, iconBg: 'bg-emerald-500/15', iconColor: 'text-emerald-400' },
    ...(isSuperAdmin ? [{ title: 'Usuários', value: usersCount, sub: 'Total cadastrados', icon: Users, iconBg: 'bg-amber-500/15', iconColor: 'text-amber-400' }] : []),
  ]

  return (
    <div className="space-y-8 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground mt-1">Bem-vindo, {user?.name}</p>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {statCards.map((stat) => (
          <Card key={stat.title} className="hover-lift hover-glow">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium text-muted-foreground">{stat.title}</p>
                <div className={`h-9 w-9 rounded-lg ${stat.iconBg} flex items-center justify-center`}>
                  <stat.icon className={`h-4 w-4 ${stat.iconColor}`} />
                </div>
              </div>
              <div className="mt-2">
                <p className="text-3xl font-bold">{stat.value}</p>
                <p className="text-xs text-muted-foreground mt-1">{stat.sub}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-6 lg:grid-cols-5">
        {/* Recent instances */}
        <Card className="lg:col-span-3">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm font-medium">Instâncias Recentes</CardTitle>
            <button onClick={() => navigate('/instances')} className="text-xs text-primary font-medium hover:text-primary/80 flex items-center gap-1 transition-colors">
              Ver todas <ArrowUpRight className="h-3 w-3" />
            </button>
          </CardHeader>
          <CardContent>
            {instances.length === 0 ? (
              <div className="text-center py-12">
                <div className="mx-auto w-12 h-12 rounded-lg bg-muted flex items-center justify-center mb-3">
                  <Smartphone className="h-6 w-6 text-muted-foreground" />
                </div>
                <p className="text-sm text-muted-foreground font-medium">Nenhuma instância cadastrada</p>
              </div>
            ) : (
              <div className="space-y-1">
                {instances.slice(0, 5).map((inst) => (
                  <div key={inst.id} className="flex items-center justify-between rounded-lg px-3 py-2.5 hover:bg-white/5 transition-colors">
                    <div className="flex items-center gap-3">
                      <div className={`h-2 w-2 rounded-full ${inst.connection_status === 'connected' ? 'bg-emerald-500' : 'bg-zinc-300 dark:bg-zinc-600'}`} />
                      <div>
                        <p className="text-sm font-medium">{inst.instance_name}</p>
                        <p className="text-xs text-muted-foreground">{inst.server_name || 'Servidor'}</p>
                      </div>
                    </div>
                    <Badge variant={inst.connection_status === 'connected' ? 'success' : 'secondary'}>
                      {inst.connection_status === 'connected' ? 'Online' : inst.connection_status === 'connecting' ? 'Conectando' : 'Offline'}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Quotas */}
        {user && (
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="text-sm font-medium">Suas Cotas</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <div>
                <div className="flex justify-between text-sm mb-2">
                  <span className="text-muted-foreground">Instâncias</span>
                  <span className="font-medium">{instances.length}<span className="text-muted-foreground">/{user.max_instances}</span></span>
                </div>
                <div className="h-1.5 rounded-full bg-muted overflow-hidden">
                  <div className="h-full rounded-full bg-primary transition-all duration-500" style={{ width: `${Math.min((instances.length / (user.max_instances || 1)) * 100, 100)}%` }} />
                </div>
              </div>
              <div>
                <div className="flex justify-between text-sm mb-2">
                  <span className="text-muted-foreground">Servidores</span>
                  <span className="font-medium">{servers.length}<span className="text-muted-foreground">/{user.max_evo_servers}</span></span>
                </div>
                <div className="h-1.5 rounded-full bg-muted overflow-hidden">
                  <div className="h-full rounded-full bg-primary transition-all duration-500" style={{ width: `${Math.min((servers.length / (user.max_evo_servers || 1)) * 100, 100)}%` }} />
                </div>
              </div>
              <div className="pt-2 border-t border-border">
                <p className="text-sm text-muted-foreground mb-3">Integrações</p>
                <div className="flex gap-2">
                  <div className={`flex-1 rounded-lg p-3 text-center border ${user.can_use_chatwoot ? 'border-emerald-500/20 bg-emerald-500/10' : 'border-border bg-muted'}`}>
                    <p className={`text-xs font-medium ${user.can_use_chatwoot ? 'text-emerald-400' : 'text-muted-foreground'}`}>Chatwoot</p>
                    <p className={`text-[11px] mt-0.5 ${user.can_use_chatwoot ? 'text-emerald-500' : 'text-muted-foreground'}`}>{user.can_use_chatwoot ? 'Ativo' : 'Inativo'}</p>
                  </div>
                  <div className={`flex-1 rounded-lg p-3 text-center border ${user.can_use_typebot ? 'border-emerald-500/20 bg-emerald-500/10' : 'border-border bg-muted'}`}>
                    <p className={`text-xs font-medium ${user.can_use_typebot ? 'text-emerald-400' : 'text-muted-foreground'}`}>Typebot</p>
                    <p className={`text-[11px] mt-0.5 ${user.can_use_typebot ? 'text-emerald-500' : 'text-muted-foreground'}`}>{user.can_use_typebot ? 'Ativo' : 'Inativo'}</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
