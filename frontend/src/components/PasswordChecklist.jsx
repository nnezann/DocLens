import { Check, X } from 'lucide-react'
import { getPasswordStrength } from '../utils/validators'

export default function PasswordChecklist({ password }) {
  const { results } = getPasswordStrength(password)

  return (
    <ul className="mt-2 space-y-1.5">
      {results.map((rule) => (
        <li
          key={rule.id}
          className={`flex items-center gap-2 text-xs ${
            rule.met ? 'text-emerald-600' : 'text-slate-400'
          }`}
        >
          {rule.met ? (
            <Check size={14} strokeWidth={2.5} className="shrink-0" />
          ) : (
            <X size={14} strokeWidth={2.5} className="shrink-0" />
          )}
          {rule.label}
        </li>
      ))}
    </ul>
  )
}