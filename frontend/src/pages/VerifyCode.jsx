import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import OtpInput from '../components/OtpInput'
import { PrimaryButton } from '../components/Button'

export default function VerifyCode() {
  const navigate = useNavigate()
  const location = useLocation()
  const [code, setCode] = useState('')

  // Where to go once the code is confirmed: password reset flow lands on
  // Reset password, signup flow lands on Account details.
  const nextPath = location.state?.nextPath ?? '/signup/account-details'
  const maskedEmail = location.state?.maskedEmail ?? 'm*******n@work.com'

  const handleSubmit = (e) => {
    e.preventDefault()
    navigate(nextPath)
  }

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Check your inbox!
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        We sent a 6-code email verification to your email
        <br />
        {maskedEmail}
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-8">
        <OtpInput onComplete={setCode} />
        <PrimaryButton type="submit" disabled={code.length !== 6}>
          Continue
        </PrimaryButton>
      </form>
    </AuthLayout>
  )
}
