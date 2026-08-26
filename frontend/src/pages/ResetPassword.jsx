import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import TextField from '../components/TextField'
import PasswordChecklist from '../components/PasswordChecklist'
import { PrimaryButton } from '../components/Button'
import { getPasswordStrength } from '../utils/validators'

export default function ResetPassword() {
  const navigate = useNavigate()
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [touched, setTouched] = useState(false)

  const { isStrong } = getPasswordStrength(password)
  const passwordsMatch = confirmPassword.length > 0 && password === confirmPassword
  const matchError = touched && confirmPassword.length > 0 && !passwordsMatch
    ? 'Passwords do not match.'
    : ''

  const handleSubmit = (e) => {
    e.preventDefault()
    setTouched(true)
    if (!isStrong || !passwordsMatch) return
    navigate('/login')
  }

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Reset password
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Choose a new password to regain access to your account.
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        <div>
          <TextField
            id="new-password"
            label="New password"
            type="password"
            placeholder="Enter Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <PasswordChecklist password={password} />
        </div>
        <TextField
          id="confirm-password"
          label="Confirm password"
          type="password"
          placeholder="Confirm Password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          onBlur={() => setTouched(true)}
          error={matchError}
          required
        />
        <PrimaryButton type="submit">Continue</PrimaryButton>
      </form>
    </AuthLayout>
  )
}