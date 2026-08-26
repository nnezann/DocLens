import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Calendar, ChevronRight, Flag, RefreshCcw, Tag, Users } from 'lucide-react'
import DashboardLayout from '../components/DashboardLayout'
import StatusBadge from '../components/StatusBadge'
import FilterDropdown from '../components/FilterDropdown'
import { INVESTIGATIONS } from '../data/investigations'

const TABS = ['All', 'Draft', 'Submitted', 'Under Review', 'Action Required', 'Completed', 'Closed']

export default function Investigations() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('All')

  const rows = useMemo(() => {
    if (activeTab === 'All') return INVESTIGATIONS
    return INVESTIGATIONS.filter((inv) => inv.status === activeTab)
  }, [activeTab])

  return (
    <DashboardLayout>
      <h1 className="font-display text-3xl font-extrabold tracking-tight text-ink-900">
        My Investigations
      </h1>

      <div className="mt-6 flex flex-wrap items-center gap-2">
        {TABS.map((tab) => {
          const isActive = tab === activeTab
          return (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={`rounded-lg px-4 py-2.5 text-sm font-medium transition ${
                isActive ? 'bg-ink-900 text-white' : 'border border-slate-200 text-ink-900 hover:bg-slate-50'
              }`}
            >
              {tab}
            </button>
          )
        })}
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <FilterDropdown icon={Calendar} label="Date" />
        <FilterDropdown icon={Tag} label="Investigation type" />
        <FilterDropdown icon={Flag} label="Priority" />
        <FilterDropdown icon={Users} label="Assigned Examiners" />
        <FilterDropdown icon={RefreshCcw} label="Clear filters" />
      </div>

      <div className="mt-6 overflow-hidden rounded-2xl border border-slate-100">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="text-slate-400">
              <th className="px-6 py-3 font-medium">Investigation</th>
              <th className="px-6 py-3 font-medium">Case ID</th>
              <th className="px-6 py-3 font-medium">Type</th>
              <th className="px-6 py-3 font-medium">Submitted</th>
              <th className="px-6 py-3 font-medium">Status</th>
              <th className="px-6 py-3 font-medium">Last Updated</th>
              <th className="px-6 py-3 font-medium">Action</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((inv) => (
              <tr
                key={inv.id}
                onClick={() => navigate(`/dashboard/investigations/${inv.id}`)}
                className="cursor-pointer border-t border-slate-100 hover:bg-slate-50"
              >
                <td className="px-6 py-4">
                  <div className="flex items-center gap-3">
                    <ChevronRight size={16} className="text-slate-300" />
                    <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-100 text-slate-500">
                      \ud83d\udcc1
                    </span>
                    <div>
                      <p className="font-medium text-ink-900">{inv.title}</p>
                      <p className="text-xs text-slate-400">{inv.fileName}</p>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 text-slate-600">{inv.caseId}</td>
                <td className="px-6 py-4 text-slate-600">{inv.type}</td>
                <td className="px-6 py-4 text-slate-500">
                  {inv.submittedDate}
                  <br />
                  {inv.submittedTime}
                </td>
                <td className="px-6 py-4">
                  <StatusBadge status={inv.status} />
                </td>
                <td className="px-6 py-4 text-slate-500">
                  {inv.lastUpdated}
                  <br />
                  by {inv.lastUpdatedBy}
                </td>
                <td className="px-6 py-4 text-slate-400" onClick={(e) => e.stopPropagation()}>
                  &bull;&bull;&bull;
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mt-4 flex items-center justify-between text-sm text-slate-500">
        <span>
          Showing 1 to {rows.length} of {INVESTIGATIONS.length} investigations
        </span>
        <div className="flex items-center gap-1">
          {['1', '2', '3', '...', '5'].map((p) => (
            <button
              key={p}
              type="button"
              className={`h-8 w-8 rounded-lg text-sm ${
                p === '1' ? 'bg-ink-900 text-white' : 'text-slate-500 hover:bg-slate-100'
              }`}
            >
              {p}
            </button>
          ))}
        </div>
      </div>
    </DashboardLayout>
  )
}