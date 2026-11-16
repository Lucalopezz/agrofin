import { useState } from 'react'
import { AuthForm } from '@/features/auth/components/auth-form'

type Props = {
  mode: 'login' | 'register'
  onNavigate: (route: 'landing' | 'login' | 'register' | 'inicio') => void
}

export function AuthPage({ mode, onNavigate }: Props) {
  const [registered, setRegistered] = useState(false)
  const registering = mode === 'register'

  return (
    <main className="auth-page">
      <section className="auth-card">
        {registered ? (
          <>
            <h1>Conta criada</h1>
            <p>Entre com seu nome de acesso e senha.</p>
            <button className="auth-submit" onClick={() => onNavigate('login')}>
              Ir para login
            </button>
          </>
        ) : (
          <>
            <a href="#/" className="auth-brand">
              Agrofin
            </a>
            <h1>{registering ? 'Criar conta' : 'Entrar'}</h1>
            <p>
              {registering
                ? 'Crie sua conta para acessar sua produção.'
                : 'Acesse sua conta para continuar.'}
            </p>
            <AuthForm
              mode={mode}
              onRegistered={() => setRegistered(true)}
              onLogin={() => onNavigate('inicio')}
            />
            <button
              className="auth-link"
              onClick={() => onNavigate(registering ? 'login' : 'register')}
            >
              {registering ? 'Já tenho conta' : 'Criar conta'}
            </button>
            <button className="auth-link" onClick={() => onNavigate('landing')}>
              Voltar à apresentação
            </button>
          </>
        )}
      </section>
    </main>
  )
}
