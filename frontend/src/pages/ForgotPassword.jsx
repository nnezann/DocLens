import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import TextField from '../components/TextField'
import { PrimaryButton } from '../components/Button'
import { isValidEmailFormat, checkEmailExists } from '../utils/validators'

export default function ForgotPassword() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [checking, setChecking] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')

    if (!isValidEmailFormat(email)) {
      setError('Enter a valid email address (e.g. name@company.com).')
      return
    }

    setChecking(true)
    const exists = await checkEmailExists(email)
    setChecking(false)

    if (!exists) {
      setError("We couldn't find an account with that email.")
      return
    }

    navigate('/login/verify', {
      state: { nextPath: '/reset-password', maskedEmail: 'm*******n@work.com' },
    })
  }

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Forgot password?
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Don&rsquo;t worry. Enter your email and we&rsquo;ll help you reset your password.
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        <TextField
          id="forgot-email"
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
        <PrimaryButton type="submit" disabled={checking}>
          {checking ? 'Checking...' : 'Continue'}
        </PrimaryButton>
      </form>
    </AuthLayout>
  )
}