import { Routes, Route, Navigate } from 'react-router-dom'
import Login from './pages/Login'
import LoginWithEmail from './pages/LoginWithEmail'
import Signup from './pages/Signup'
import OrganizationSetup from './pages/OrganizationSetup'
import ForgotPassword from './pages/ForgotPassword'
import VerifyCode from './pages/VerifyCode'
import ResetPassword from './pages/ResetPassword'
import AccountDetails from './pages/AccountDetails'
import AccountSecurity from './pages/AccountSecurity'
import OrganizationData from './pages/OrganizationData'
import Dashboard from './pages/Dashboard'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />

      {/* Login flow */}
      <Route path="/login" element={<Login />} />
      <Route path="/login/email" element={<LoginWithEmail />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/login/verify" element={<VerifyCode />} />
      <Route path="/reset-password" element={<ResetPassword />} />

      {/* Signup flow */}
      <Route path="/signup" element={<Signup />} />
      <Route path="/signup/work-email" element={<OrganizationSetup />} />
      <Route path="/signup/verify" element={<VerifyCode />} />
      <Route path="/signup/account-details" element={<AccountDetails />} />
      <Route path="/signup/account-security" element={<AccountSecurity />} />
      <Route path="/signup/organization-data" element={<OrganizationData />} />

      {/* App */}
      <Route path="/dashboard" element={<Dashboard />} />

      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}
