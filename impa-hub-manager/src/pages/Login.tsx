import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'
import { Network, Eye, EyeOff, ArrowRight, Zap } from 'lucide-react'
import { Button, Input, Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui'
import { authApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'

const loginSchema = z.object({
  email: z.string().email('Email inválido'),
  password: z.string().min(1, 'Senha é obrigatória'),
})

type LoginForm = z.infer<typeof loginSchema>

export default function LoginPage() {
  const navigate = useNavigate()
  const setAuth = useAuthStore((s) => s.setAuth)
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
  })

  const onSubmit = async (data: LoginForm) => {
    setLoading(true)
    try {
      const res = await authApi.login(data)
      setAuth(res.token, res.user)
      toast.success('Login realizado com sucesso!')
      navigate('/')
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Erro ao fazer login')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen">
      {/* Left panel - branding */}
      <div className="hidden lg:flex lg:w-[55%] bg-gradient-to-br from-sidebar via-sidebar to-sidebar/80 items-center justify-center relative overflow-hidden">
        {/* Decorative gradient orbs */}
        <div className="absolute top-1/4 right-1/4 w-96 h-96 bg-primary/10 rounded-full blur-3xl" />
        <div className="absolute bottom-1/4 left-1/4 w-64 h-64 bg-cyan-500/10 rounded-full blur-3xl" />
        
        <div className="relative z-10 text-center space-y-8 px-16 max-w-lg animate-fade-in">
          <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-primary/70 shadow-2xl shadow-primary/30">
            <Zap className="h-10 w-10 text-primary-foreground" />
          </div>
          <div className="space-y-3">
            <h1 className="text-4xl font-bold text-sidebar-foreground tracking-tight">IMPA HUB</h1>
            <p className="text-base text-sidebar-foreground/50 leading-relaxed max-w-md mx-auto">
              Gerencie suas instâncias WhatsApp, integrações e automações em um único painel.
            </p>
          </div>
          <div className="flex items-center justify-center gap-4 pt-4">
            <div className="glass-sm px-4 py-2 text-xs font-medium text-sidebar-foreground/70">WhatsApp</div>
            <div className="glass-sm px-4 py-2 text-xs font-medium text-sidebar-foreground/70">Chatwoot</div>
            <div className="glass-sm px-4 py-2 text-xs font-medium text-sidebar-foreground/70">Typebot</div>
          </div>
        </div>
      </div>

      {/* Right panel - login form */}
      <div className="flex flex-1 items-center justify-center bg-background p-8">
        <div className="w-full max-w-[380px] space-y-10 animate-slide-up">
          <div className="text-center lg:text-left space-y-2">
            <div className="lg:hidden mx-auto mb-8 flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary/70 shadow-lg shadow-primary/20">
              <Zap className="h-7 w-7 text-primary-foreground" />
            </div>
            <h2 className="text-3xl font-bold tracking-tight">Bem-vindo</h2>
            <p className="text-muted-foreground">Faça login para acessar o painel</p>
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
            <Input
              id="email"
              type="email"
              label="Email"
              placeholder="seu@email.com"
              error={errors.email?.message}
              {...register('email')}
            />
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                label="Senha"
                placeholder="••••••••"
                error={errors.password?.message}
                {...register('password')}
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3.5 top-9 text-muted-foreground hover:text-foreground transition-colors"
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            <Button type="submit" className="w-full h-11 text-sm" loading={loading}>
              Entrar <ArrowRight className="h-4 w-4" />
            </Button>
          </form>

          <p className="text-center text-[11px] text-muted-foreground font-medium">
            &copy; {new Date().getFullYear()} IMPA HUB &middot; Integration Hub
          </p>
        </div>
      </div>
    </div>
  )
}
