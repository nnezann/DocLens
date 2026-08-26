const PRIORITY_STYLES = {
  Normal: 'bg-slate-100 text-slate-600',
  High: 'bg-orange-50 text-orange-700',
  Urgent: 'bg-red-50 text-red-700',
}

export default function PriorityBadge({ priority }) {
  const style = PRIORITY_STYLES[priority] ?? 'bg-slate-100 text-slate-600'
  return (
    <span className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-medium ${style}`}>
      {priority}
    </span>
  )
}