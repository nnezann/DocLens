import { useState } from 'react'
import { ChevronRight } from 'lucide-react'
import DashboardLayout from '../components/DashboardLayout'
import TextField from '../components/TextField'
import SelectField from '../components/SelectField'
import { PrimaryButton, SecondaryButton } from '../components/Button'

const PRIORITIES = ['Normal', 'High', 'Urgent']

export default function Dashboard() {
  const [priority, setPriority] = useState('High')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')

  const handleSubmit = (e) => {
    e.preventDefault()
  }

  return (
    <DashboardLayout>
      <nav className="flex items-center gap-1.5 text-sm text-slate-400">
        <span>Investigations</span>
        <ChevronRight size={14} />
        <span className="text-ink-900">Add investigation</span>
      </nav>

      <h1 className="mt-3 font-display text-4xl font-extrabold tracking-tight text-ink-900">
        Add New Case
      </h1>

      <form onSubmit={handleSubmit} className="mt-8 max-w-2xl">
        <div className="mb-8 flex items-center gap-3">
          <span className="flex h-8 w-8 items-center justify-center rounded-full bg-accent text-sm font-semibold text-white">
            1
          </span>
          <h2 className="text-lg font-semibold text-ink-900">Basic Information</h2>
        </div>

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
          <div>
            <p className="mb-2 text-sm font-medium text-ink-900">Case ID</p>
            <TextField id="case-id" label="Case ID" value="FDE-202464" disabled className="text-slate-400" />
          </div>
          <div>
            <p className="mb-2 text-sm font-medium text-ink-900">Case Type</p>
            <SelectField id="case-type" label="Case Type" defaultValue="signature-verification">
              <option value="signature-verification">Signature verification</option>
              <option value="document-authenticity">Document authenticity</option>
              <option value="fraud-review">Fraud review</option>
            </SelectField>
          </div>
        </div>

        <div className="mt-6">
          <p className="mb-2 text-sm font-medium text-ink-900">Investigation Title</p>
          <TextField
            id="investigation-title"
            label="Investigation Title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
          />
        </div>

        <div className="mt-6">
          <p className="mb-2 text-sm font-medium text-ink-900">Investigation Description</p>
          <textarea
            id="investigation-description"
            placeholder="Type here..."
            rows={5}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full resize-none rounded-xl border border-slate-300 bg-white px-4 py-3.5 text-[15px] text-ink-900 placeholder:text-slate-400 focus:border-ink-900 focus:outline-none focus:ring-1 focus:ring-ink-900"
          />
        </div>

        <fieldset className="mt-6">
          <legend className="mb-3 text-sm font-medium text-ink-900">Priority</legend>
          <div className="flex items-center gap-8">
            {PRIORITIES.map((option) => (
              <label key={option} className="flex cursor-pointer items-center gap-2 text-[15px] text-ink-900">
                <input
                  type="radio"
                  name="priority"
                  value={option}
                  checked={priority === option}
                  onChange={() => setPriority(option)}
                  className="h-4 w-4 accent-ink-900"
                />
                {option}
              </label>
            ))}
          </div>
        </fieldset>

        <div className="mt-10 flex items-center gap-4">
          <SecondaryButton type="button" className="w-auto px-8">
            Cancel
          </SecondaryButton>
          <PrimaryButton type="submit" className="w-auto px-8">
            Save &amp; Continue
          </PrimaryButton>
        </div>
      </form>
    </DashboardLayout>
  )
}
