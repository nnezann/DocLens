import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthLayout from '../components/AuthLayout'
import TextField from '../components/TextField'
import SelectField from '../components/SelectField'
import { PrimaryButton } from '../components/Button'

export default function OrganizationData() {
  const navigate = useNavigate()
  const [orgName, setOrgName] = useState('')

  const handleSubmit = (e) => {
    e.preventDefault()
    navigate('/dashboard')
  }

  return (
    <AuthLayout>
      <h1 className="font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Organization data
      </h1>
      <p className="mt-3 text-center text-[15px] leading-relaxed text-slate-500">
        Let&rsquo;s setup your organization. Enter details to setup your workspace
      </p>

      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        <TextField
          id="org-name"
          label="Organization name"
          placeholder="Organization name"
          value={orgName}
          onChange={(e) => setOrgName(e.target.value)}
          required
        />
        <SelectField id="org-type" label="Organization type" defaultValue="">
          <option value="" disabled>
            Select Organization type
          </option>
          <option value="law-firm">Law firm</option>
          <option value="corporate">Corporate legal</option>
          <option value="government">Government</option>
          <option value="other">Other</option>
        </SelectField>
        <SelectField id="org-size" label="Organization size" defaultValue="">
          <option value="" disabled>
            Organization Size
          </option>
          <option value="1-10">1–10 employees</option>
          <option value="11-50">11–50 employees</option>
          <option value="51-200">51–200 employees</option>
          <option value="200+">200+ employees</option>
        </SelectField>
        <SelectField id="org-country" label="Country" defaultValue="">
          <option value="" disabled>
            Country
          </option>
          <option value="rw">Rwanda</option>
          <option value="us">United States</option>
          <option value="uk">United Kingdom</option>
          <option value="other">Other</option>
        </SelectField>
        <PrimaryButton type="submit">Continue</PrimaryButton>
      </form>
    </AuthLayout>
  )
}
