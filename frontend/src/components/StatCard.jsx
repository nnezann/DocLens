export default function StatCard({ icon: Icon, label, value, footnote, trend }) {
  return (
    <div className="rounded-2xl border border-slate-100 p-5">
      <div className="flex items-center gap-2 text-sm text-slate-500">
        <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-100 text-ink-900">
          <Icon size={16} strokeWidth={1.8} />
        </span>
        {label}
      </div>
      <p className="mt-4 font-display text-3xl font-bold text-ink-900">{value}</p>
      {footnote && (
        <p className="mt-1 flex items-center gap-1 text-xs text-slate-400">
          {trend && <span className="text-emerald-600">{trend}</span>}
          {footnote}
        </p>
      )}
    </div>
  )
}