export default function Tabs({ tabs, active, onChange }) {
  return (
    <div className="flex gap-6 border-b border-slate-100">
      {tabs.map((tab) => {
        const isActive = tab === active
        return (
          <button
            key={tab}
            type="button"
            onClick={() => onChange(tab)}
            className={`-mb-px border-b-2 pb-3 text-[15px] font-medium transition ${
              isActive
                ? 'border-ink-900 text-ink-900'
                : 'border-transparent text-slate-400 hover:text-ink-900'
            }`}
          >
            {tab}
          </button>
        )
      })}
    </div>
  )
}