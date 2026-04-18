import { useState } from 'react'
import { toast } from 'sonner'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Key, Sun, Moon, User, Globe, ShieldCheck } from 'lucide-react'
import { Button, Input, Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui'
import { authApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useTheme } from '@/contexts/ThemeContext'

const passwordSchema = z.object({
  current_password: z.string().min(1, 'Senha atual é obrigatória'),
  new_password: z.string().min(6, 'Mínimo 6 caracteres'),
  confirm_password: z.string().min(1, 'Confirme a senha'),
}).refine(d => d.new_password === d.confirm_password, {
  message: 'Senhas não conferem',
  path: ['confirm_password'],
})

type PasswordForm = z.infer<typeof passwordSchema>

export default function SettingsPage() {
  const { user } = useAuthStore()
  const { theme, toggleTheme } = useTheme()
  const [submitting, setSubmitting] = useState(false)
  const [apiUrl, setApiUrl] = useState(
    (window as any).__ENV__?.VITE_API_URL || import.meta.env.VITE_API_URL || 'http://localhost:4050/api/v1'
  )

  const { register, handleSubmit, reset, formState: { errors } } = useForm<PasswordForm>({
    resolver: zodResolver(passwordSchema),
  })

  const onChangePassword = async (data: PasswordForm) => {
    setSubmitting(true)
    try {
      await authApi.changePassword({
        current_password: data.current_password,
        new_password: data.new_password,
      })
      toast.success('Senha alterada com sucesso!')
      reset()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao alterar senha')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-6 max-w-2xl animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Configurações</h1>
        <p className="text-muted-foreground mt-1">Gerencie suas preferências e segurança</p>
      </div>

      {/* Profile */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-md bg-muted flex items-center justify-center">
              <User className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <CardTitle className="text-base font-medium">Perfil</CardTitle>
              <CardDescription className="text-muted-foreground">Informações da sua conta</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 rounded-lg bg-muted p-4">
            <div>
              <p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider mb-1">Nome</p>
              <p className="font-medium">{user?.name}</p>
            </div>
            <div>
              <p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider mb-1">Email</p>
              <p className="font-semibold">{user?.email}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Theme */}
      <Card className="">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-md bg-muted flex items-center justify-center">
              <Sun className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <CardTitle className="text-base font-medium">Aparência</CardTitle>
              <CardDescription className="text-muted-foreground">Personalize a interface</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex gap-3">
            <button
              onClick={() => theme !== 'light' && toggleTheme()}
              className={`flex-1 flex items-center justify-center gap-2 rounded-md py-3 text-sm font-medium transition-colors ${theme === 'light' ? 'bg-primary text-primary-foreground' : 'bg-muted hover:bg-muted/80'}`}
            >
              <Sun className="h-4 w-4" /> Claro
            </button>
            <button
              onClick={() => theme !== 'dark' && toggleTheme()}
              className={`flex-1 flex items-center justify-center gap-2 rounded-md py-3 text-sm font-medium transition-colors ${theme === 'dark' ? 'bg-primary text-primary-foreground' : 'bg-muted hover:bg-muted/80'}`}
            >
              <Moon className="h-4 w-4" /> Escuro
            </button>
          </div>
        </CardContent>
      </Card>

      {/* Change Password */}
      <Card className="">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-md bg-muted flex items-center justify-center">
              <ShieldCheck className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <CardTitle className="text-base font-medium">Alterar Senha</CardTitle>
              <CardDescription className="text-muted-foreground">Atualize sua senha de acesso</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onChangePassword)} className="space-y-4">
            <Input
              id="current_password"
              type="password"
              label="Senha Atual"
              error={errors.current_password?.message}
              {...register('current_password')}
            />
            <Input
              id="new_password"
              type="password"
              label="Nova Senha"
              error={errors.new_password?.message}
              {...register('new_password')}
            />
            <Input
              id="confirm_password"
              type="password"
              label="Confirmar Nova Senha"
              error={errors.confirm_password?.message}
              {...register('confirm_password')}
            />
            <Button type="submit" loading={submitting}>
              <Key className="h-4 w-4" /> Alterar Senha
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* API URL */}
      <Card className="">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-md bg-muted flex items-center justify-center">
              <Globe className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <CardTitle className="text-base font-medium">Conexão API</CardTitle>
              <CardDescription className="text-muted-foreground">URL do backend IMPA HUB</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-sm font-mono bg-muted rounded-lg px-4 py-3 break-all text-foreground">{apiUrl}</p>
          <p className="text-xs text-muted-foreground mt-2">Configure via variável de ambiente VITE_API_URL</p>
        </CardContent>
      </Card>
    </div>
  )
}
