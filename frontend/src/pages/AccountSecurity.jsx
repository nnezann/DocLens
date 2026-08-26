import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import TextField from '../components/TextField'
import { PrimaryButton } from '../components/Button'

export default function AccountSecurity() {
  const navigate = useNavigate()
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  const handleSubmit = (e) => {
    e.preventDefault()
    navigate('/signup/organization-data')
  }

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Account Security
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Let&rsquo;s secure your account. Enter your password to protect your account
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        <TextField
          id="account-password"
          label="Password"
          type="password"
          placeholder="Enter Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        <TextField
          id="account-confirm-password"
          label="Confirm password"
          type="password"
          placeholder="Confirm Password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          required
        />
        <PrimaryButton type="submit">Continue</PrimaryButton>
      </form>
    </AuthLayout>
  )
}
