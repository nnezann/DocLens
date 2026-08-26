import { Link, useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import { PrimaryButton, SecondaryButton } from '../components/Button'
import GoogleIcon from '../components/GoogleIcon'

export default function Signup() {
  const navigate = useNavigate()

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Create an Account
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Set up your account to securely analyze and manage your documents.
      </p>

      <SecondaryButton className="mt-8 flex items-center justify-center gap-3">
        <GoogleIcon />
        Continue with Google
      </SecondaryButton>

      <div className="my-6 flex items-center gap-4">
        <div className="h-px flex-1 bg-slate-200" />
        <span className="text-sm text-slate-400">OR</span>
        <div className="h-px flex-1 bg-slate-200" />
      </div>

      <PrimaryButton onClick={() => navigate('/signup/work-email')}>
        Sign up with email
      </PrimaryButton>

      <p className="mt-5 text-center text-sm leading-relaxed text-slate-500">
        By signing up, you agree to the{' '}
        <a href="#" className="font-medium text-ink-900 underline underline-offset-2">
          Terms of Service
        </a>{' '}
        and{' '}
        <a href="#" className="font-medium text-ink-900 underline underline-offset-2">
          Privacy Policy,
        </a>{' '}
        including{' '}
        <a href="#" className="font-medium text-ink-900 underline underline-offset-2">
          cookie use.
        </a>
      </p>

      <p className="mt-16 text-center text-sm text-slate-600">
        Already have an ccount?{' '}
        <Link to="/login" className="font-medium text-ink-900 underline underline-offset-2">
          Log in
        </Link>
      </p>
    </AuthLayout>
  )
}
