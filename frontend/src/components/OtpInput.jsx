import { useRef, useState } from 'react'

export default function OtpInput({ length = 6, onComplete }) {
  const [values, setValues] = useState(Array(length).fill(''))
  const inputsRef = useRef([])

  const handleChange = (index, raw) => {
    const digit = raw.replace(/\D/g, '').slice(-1)
    const next = [...values]
    next[index] = digit
    setValues(next)

    if (digit && index < length - 1) {
      inputsRef.current[index + 1]?.focus()
    }
    if (next.every((v) => v !== '')) {
      onComplete?.(next.join(''))
    }
  }

  const handleKeyDown = (index, e) => {
    if (e.key === 'Backspace' && !values[index] && index > 0) {
      inputsRef.current[index - 1]?.focus()
    }
  }

  const handlePaste = (e) => {
    const pasted = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, length)
    if (!pasted) return
    e.preventDefault()
    const next = Array(length).fill('')
    pasted.split('').forEach((char, i) => (next[i] = char))
    setValues(next)
    const focusIndex = Math.min(pasted.length, length - 1)
    inputsRef.current[focusIndex]?.focus()
    if (pasted.length === length) onComplete?.(pasted)
  }

  return (
    <div className="flex gap-3" onPaste={handlePaste}>
      {values.map((value, i) => (
        <input
          key={i}
          ref={(el) => (inputsRef.current[i] = el)}
          value={value}
          onChange={(e) => handleChange(i, e.target.value)}
          onKeyDown={(e) => handleKeyDown(i, e)}
          inputMode="numeric"
          maxLength={1}
          aria-label={`Digit ${i + 1} of verification code`}
          className="h-14 w-12 rounded-xl border border-slate-300 text-center text-lg font-medium text-ink-900 focus:border-ink-900 focus:outline-none focus:ring-1 focus:ring-ink-900"
        />
      ))}
    </div>
  )
}
