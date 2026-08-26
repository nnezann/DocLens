import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import TextField from '../components/TextField'
import { PrimaryButton } from '../components/Button'
import { isValidEmailFormat } from '../utils/validators'

export default function OrganizationSetup() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = (e) => {
    e.preventDefault()

    if (!isValidEmailFormat(email)) {
      setError('Enter a valid email address (e.g. name@company.com).')
      return
    }

    setError('')
    navigate('/signup/verify')
  }

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Organization Setup
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Enter your work email to get started.
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        <TextField
          id="work-email"
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
        <PrimaryButton type="submit">Continue</PrimaryButton>
      </form>

      <p className="mt-6 text-center text-sm text-slate-600">
        Already have an account?{' '}
        <Link to="/login" className="font-medium text-ink-900 underline underline-offset-2">
          Log in
        </Link>
      </p>
    </AuthLayout>
  )
}