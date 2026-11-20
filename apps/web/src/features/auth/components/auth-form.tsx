import { useState, type FormEvent } from 'react'
import { AuthField } from '@/features/auth/components/auth-field'
import { useRegister } from '@/features/auth/hooks/use-register'
import { useSession } from '@/features/auth/session-context'
import { ApiError } from '@/shared/lib/api-client'

type AuthFormProps = {
  mode: 'login' | 'register'
  onRegistered: () => void
  onLogin: () => void
}

export function AuthForm({ mode, onRegistered, onLogin }: AuthFormProps) {
  const { login } = useSession()
  const register = useRegister()
  const [name, setName] = useState('')
  const [loginName, setLoginName] = useState('')
  const [password, setPassword] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const registering = mode === 'register'

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setError('')
    if (registering && password !== confirmation) {
      setError('As senhas não coincidem.')
      return
    }
    setBusy(true)
    try {
      if (registering) {
        await register({ name, login_name: loginName, password })
        setPassword('')
        setConfirmation('')
        onRegistered()
      } else {
        await login({ login_name: loginName, password })
        onLogin()
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 401)
        setError('Nome de acesso ou senha inválidos.')
      else if (err instanceof ApiError && err.status === 409)
        setError('Este nome de acesso já está em uso.')
      else if (err instanceof ApiError && err.status === 400)
        setError(err.message.charAt(0).toUpperCase() + err.message.slice(1))
      else setError('Não foi possível concluir. Confira a conexão e tente novamente.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={submit}>
      {registering && (
        <AuthField
          label="Nome"
          autoComplete="name"
          required
          maxLength={100}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      )}
      <AuthField
        label="Nome de acesso"
        autoComplete="username"
        required
        value={loginName}
        onChange={(e) => setLoginName(e.target.value)}
      />
      <AuthField
        label="Senha"
        type="password"
        autoComplete={registering ? 'new-password' : 'current-password'}
        required
        value={password}
        onChange={(e) => setPassword(e.target.value)}
      />
      {registering && (
        <>
          <small>Use uma senha com 6 a 128 caracteres.</small>
          <AuthField
            label="Confirmar senha"
            type="password"
            autoComplete="new-password"
            required
            value={confirmation}
            onChange={(e) => setConfirmation(e.target.value)}
          />
        </>
      )}
      {error && (
        <p className="auth-error" role="alert">
          {error}
        </p>
      )}
      <button className="auth-submit" disabled={busy} type="submit">
        {busy ? 'Aguarde...' : registering ? 'Criar conta' : 'Entrar'}
      </button>
    </form>
  )
}
