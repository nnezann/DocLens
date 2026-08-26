export function PrimaryButton({ children, className = '', type = 'button', ...props }) {
  return (
    <button
      type={type}
      className={`w-full rounded-xl bg-ink-900 py-3.5 text-[15px] font-medium text-white transition hover:bg-ink-800 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${className}`}
      {...props}
    >
      {children}
    </button>
  )
}

export function SecondaryButton({ children, className = '', type = 'button', ...props }) {
  return (
    <button
      type={type}
      className={`w-full rounded-xl border border-slate-300 bg-white py-3.5 text-[15px] font-medium text-ink-900 transition hover:bg-slate-50 ${className}`}
      {...props}
    >
      {children}
    </button>
  )
}
