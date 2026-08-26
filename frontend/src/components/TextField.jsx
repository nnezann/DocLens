export default function TextField({ label, id, error, className = '', ...props }) {
  return (
    <div className={className}>
      {label && (
        <label htmlFor={id} className="sr-only">
          {label}
        </label>
      )}
      <input
        id={id}
        aria-invalid={!!error}
        className={`w-full rounded-xl border bg-white px-4 py-3.5 text-[15px] text-ink-900 placeholder:text-slate-400 focus:outline-none focus:ring-1 ${
          error
            ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
            : 'border-slate-300 focus:border-ink-900 focus:ring-ink-900'
        }`}
        {...props}
      />
      {error && <p className="mt-1.5 text-xs text-red-500">{error}</p>}
    </div>
  )
}