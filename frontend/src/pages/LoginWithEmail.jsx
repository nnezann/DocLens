import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import TextField from '../components/TextField'
import { PrimaryButton } from '../components/Button'
import { isValidEmailFormat } from '../utils/validators'

export default function LoginWithEmail() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = (e) => {
    e.preventDefault()

    if (!isValidEmailFormat(email)) {
      setError('Enter a valid email address (e.g. name@company.com).')
      return
    }

    // Real auth: the backend should return one generic "invalid email or
    // password" error here rather than confirming whether the email
    // exists — that avoids leaking which emails are registered.
    setError('')
    navigate('/dashboard')
  }

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Welcome Back!
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Log in into account to securely analyze and manage your documents.
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        <TextField
          id="login-email"
          label="Work email"
          type="email"
          placeholder="Enter work email"
          value={email}
          onChange={(e) => {
            setEmail(e.target.value)
            if (error) setError('')
          }}
          error={error}
          required
        />
        <div>
          <TextField
            id="login-password"
            label="Password"
            type="password"
            placeholder="Enter Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <div className="mt-2 text-right">
            <Link
              to="/forgot-password"
              className="text-sm font-medium text-ink-900 underline underline-offset-2"
            >
              Forgot password?
            </Link>
          </div>
        </div>
        <PrimaryButton type="submit">Continue</PrimaryButton>
      </form>

      <p className="mt-6 text-center text-sm text-slate-600">
        Don&rsquo;t have account?{' '}
        <Link to="/signup" className="font-medium text-ink-900 underline underline-offset-2">
          Signup
        </Link>
      </p>
    </AuthLayout>
  )
}