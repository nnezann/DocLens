export default function TextField({ label, id, className = '', ...props }) {
  return (
    <div className={className}>
      {label && (
        <label htmlFor={id} className="sr-only">
          {label}
        </label>
      )}
      <input
        id={id}
        className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3.5 text-[15px] text-ink-900 placeholder:text-slate-400 focus:border-ink-900 focus:outline-none focus:ring-1 focus:ring-ink-900"
        {...props}
      />
    </div>
  )
}
