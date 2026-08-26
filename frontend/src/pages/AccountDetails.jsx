import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import TextField from '../components/TextField'
import { PrimaryButton } from '../components/Button'

export default function AccountDetails() {
  const navigate = useNavigate()
  const [form, setForm] = useState({ firstName: '', lastName: '', jobTitle: '' })

  const update = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }))

  const handleSubmit = (e) => {
    e.preventDefault()
    navigate('/signup/account-security')
  }

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Account details
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Let&rsquo;s get to know you. Enter your details to create an account
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        <TextField
          id="first-name"
          label="First name"
          placeholder="First name"
          value={form.firstName}
          onChange={update('firstName')}
          required
        />
        <TextField
          id="last-name"
          label="Last name"
          placeholder="Last name"
          value={form.lastName}
          onChange={update('lastName')}
          required
        />
        <TextField
          id="job-title"
          label="Job title"
          placeholder="Job Title"
          value={form.jobTitle}
          onChange={update('jobTitle')}
          required
        />
        <PrimaryButton type="submit">Continue</PrimaryButton>
      </form>
    </AuthLayout>
  )
}
