import { useNavigate } from 'react-router-dom'
import {
  ArrowRight,
  ClipboardList,
  History,
  Plus,
  ShieldAlert,
  UploadCloud,
  UserPlus,
  CheckCircle2,
} from 'lucide-react'
import DashboardLayout from '../components/DashboardLayout'
import StatCard from '../components/StatCard'
import StatusBadge from '../components/StatusBadge'
import { PrimaryButton } from '../components/Button'
import { INVESTIGATIONS } from '../data/investigations'

const STATS = [
  { icon: ClipboardList, label: 'Active investigations', value: 12, footnote: 'this week', trend: '+3' },
  {
    icon: History,
    label: 'Awaiting review',
    value: 4,
    footnote: 'Documents are currently being examined',
  },
  { icon: CheckCircle2, label: 'Complete', value: 28, footnote: 'Completed investigations' },
  { icon: ShieldAlert, label: 'Requires attention', value: 2, footnote: 'Action required' },
]

const QUICK_ACTIONS = [
  {
    icon: ClipboardList,
    title: 'Start a new case',
    description: 'Create a new investigation from scratch',
    to: '/dashboard/investigations/new',
  },
  {
    icon: UploadCloud,
    title: 'Upload a document',
    description: 'Upload documents for analysis',
    to: '/dashboard/investigations/new',
  },
  {
    icon: History,
    title: 'View Investigation History',
    description: 'Browser and filter your investigations.',
    to: '/dashboard/investigations',
  },
  {
    icon: UserPlus,
    title: 'Invite team member',
    description: 'Add colleagues to your workspace',
    to: '/dashboard/settings',
  },
]

// First four rows, relabelled to match the "Recent Investigations" mock
// (short INV ids, an "Assigned To" column) independent of the fuller
// My Investigations table.
const RECENT = [
  { id: 'INV-2025-410', doc: 'Contract_signed.pdf', submitted: 'May 23, 2025 10:24 AM', status: 'Submitted', assignedTo: 'UWERA RUKUNDO' },
  { id: 'INV-2025-409', doc: 'Agreement_change.pdf', submitted: 'May 23, 2025 10:24 AM', status: 'Under Review', assignedTo: 'KIZERE Hope' },
  { id: 'INV-2025-408', doc: 'Invoice_Jan2025.pdf', submitted: 'May 23, 2025 10:24 AM', status: 'Action Required', assignedTo: 'Chaste GANZA' },
  { id: 'INV-2025-407', doc: 'NDA_signed.pdf', submitted: 'May 23, 2025 10:24 AM', status: 'Closed', assignedTo: 'RUGERO Kalisa' },
]

export default function Home() {
  const navigate = useNavigate()

  return (
    <DashboardLayout>
      <div className="flex items-start justify-between">
        <h1 className="font-display text-3xl font-extrabold tracking-tight text-ink-900">
          Good Morning, Seraphin
        </h1>
        <PrimaryButton
          className="flex w-auto items-center gap-2 px-5"
          onClick={() => navigate('/dashboard/investigations/new')}
        >
          <Plus size={17} strokeWidth={2.2} />
          New investigation
        </PrimaryButton>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {STATS.map((stat) => (
          <StatCard key={stat.label} {...stat} />
        ))}
      </div>

      <section className="mt-8 rounded-2xl border border-slate-100">
        <div className="flex items-center justify-between px-6 py-5">
          <h2 className="text-lg font-semibold text-ink-900">Recent Investigations</h2>
          <button
            type="button"
            onClick={() => navigate('/dashboard/investigations')}
            className="flex items-center gap-1.5 text-sm font-medium text-ink-900"
          >
            View all
            <ArrowRight size={15} />
          </button>
        </div>
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-t border-slate-100 text-slate-400">
              <th className="px-6 py-3 font-medium">Investigation</th>
              <th className="px-6 py-3 font-medium">Document</th>
              <th className="px-6 py-3 font-medium">Submitted</th>
              <th className="px-6 py-3 font-medium">Status</th>
              <th className="px-6 py-3 font-medium">Assigned To</th>
              <th className="px-6 py-3 font-medium">Action</th>
            </tr>
          </thead>
          <tbody>
            {RECENT.map((row) => (
              <tr key={row.id} className="border-t border-slate-100">
                <td className="px-6 py-4 font-medium text-ink-900">{row.id}</td>
                <td className="px-6 py-4 text-slate-500">{row.doc}</td>
                <td className="px-6 py-4 text-slate-500">{row.submitted}</td>
                <td className="px-6 py-4">
                  <StatusBadge status={row.status} />
                </td>
                <td className="px-6 py-4 text-slate-500">{row.assignedTo}</td>
                <td className="px-6 py-4 text-slate-400">&bull;&bull;&bull;</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="mt-8">
        <h2 className="mb-4 text-lg font-semibold text-ink-900">Quick Actions</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {QUICK_ACTIONS.map((action) => (
            <button
              key={action.title}
              type="button"
              onClick={() => navigate(action.to)}
              className="flex flex-col items-start rounded-2xl border border-slate-100 p-5 text-left transition hover:border-slate-200 hover:shadow-sm"
            >
              <action.icon size={20} strokeWidth={1.8} className="text-ink-900" />
              <p className="mt-3 font-semibold text-ink-900">{action.title}</p>
              <p className="mt-1 text-sm text-slate-500">{action.description}</p>
              <ArrowRight size={16} className="mt-4 text-slate-400" />
            </button>
          ))}
        </div>
      </section>
    </DashboardLayout>
  )
}