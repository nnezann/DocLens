// Half-circle gauge going from "Low" (left) to "High" (right).
// `value` is 0-1, where 0 is fully Low risk and 1 is fully High risk.
const RISK_TO_VALUE = { Low: 0.15, Moderate: 0.5, High: 0.85 }

export default function ForgeryRiskGauge({ risk = 'Low' }) {
  const value = RISK_TO_VALUE[risk] ?? 0.15
  const angle = Math.PI * (1 - value) // 0 -> right edge, PI -> left edge
  const cx = 60
  const cy = 58
  const r = 46
  const needleX = cx + r * Math.cos(angle)
  const needleY = cy - r * Math.sin(angle)

  return (
    <div className="flex flex-col items-center">
      <svg viewBox="0 0 120 70" className="w-40">
        <path
          d="M 14 58 A 46 46 0 0 1 106 58"
          fill="none"
          stroke="#E2E8F0"
          strokeWidth="9"
          strokeLinecap="round"
        />
        <path
          d="M 14 58 A 46 46 0 0 1 106 58"
          fill="none"
          stroke="#131A2C"
          strokeWidth="9"
          strokeLinecap="round"
          strokeDasharray={`${value * 145} 300`}
        />
        <line x1={cx} y1={cy} x2={needleX} y2={needleY} stroke="#131A2C" strokeWidth="2" />
        <circle cx={cx} cy={cy} r="3.5" fill="#131A2C" />
      </svg>
      <div className="-mt-1 flex w-40 justify-between text-xs text-slate-400">
        <span>Low</span>
        <span>High</span>
      </div>
      <p className="mt-2 text-2xl font-bold text-ink-900">{risk}</p>
      <p className="text-xs text-slate-400">Risk Level</p>
    </div>
  )
}