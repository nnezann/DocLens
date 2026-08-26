import { Link, useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import { PrimaryButton, SecondaryButton } from '../components/Button'
import GoogleIcon from '../components/GoogleIcon'

export default function Login() {
  const navigate = useNavigate()

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Welcome Back!
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Log in into account to securely analyze and manage your documents.
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

      <PrimaryButton onClick={() => navigate('/login/email')}>Login with email</PrimaryButton>

      <p className="mt-6 text-center text-sm text-slate-600">
        Don&rsquo;t have account?{' '}
        <Link to="/signup" className="font-medium text-ink-900 underline underline-offset-2">
          Signup
        </Link>
      </p>
    </AuthLayout>
  )
}
