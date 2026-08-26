import { ChevronDown } from 'lucide-react'

export default function FilterDropdown({ icon: Icon, label }) {
  return (
    <button
      type="button"
      className="flex items-center gap-2 rounded-lg border border-slate-200 px-4 py-2.5 text-sm font-medium text-ink-900 hover:bg-slate-50"
    >
      {Icon && <Icon size={16} strokeWidth={1.8} className="text-slate-500" />}
      {label}
      <ChevronDown size={15} strokeWidth={1.8} className="text-slate-400" />
    </button>
  )
}