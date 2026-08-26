import { ChevronDown } from 'lucide-react'

export default function SelectField({ label, id, children, className = '', ...props }) {
  return (
    <div className={`relative ${className}`}>
      {label && (
        <label htmlFor={id} className="sr-only">
          {label}
        </label>
      )}
      <select
        id={id}
        className="w-full appearance-none rounded-xl border border-slate-300 bg-white px-4 py-3.5 text-[15px] text-slate-400 focus:border-ink-900 focus:outline-none focus:ring-1 focus:ring-ink-900"
        defaultValue=""
        {...props}
      >
        {children}
      </select>
      <ChevronDown
        size={18}
        strokeWidth={2}
        className="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 text-ink-900"
      />
    </div>
  )
}
