const STATUS_STYLES = {
  Draft: 'bg-slate-100 text-slate-600',
  Submitted: 'bg-blue-50 text-blue-700',
  'Under Review': 'bg-amber-50 text-amber-700',
  'Action Required': 'bg-red-50 text-red-700',
  Completed: 'bg-emerald-50 text-emerald-700',
  Closed: 'bg-slate-100 text-slate-500',
}

export default function StatusBadge({ status }) {
  const style = STATUS_STYLES[status] ?? 'bg-slate-100 text-slate-600'
  return (
    <span className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-medium ${style}`}>
      {status}
    </span>
  )
}