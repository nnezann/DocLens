export default function SignatureSquiggle({ seed = 0 }) {
  // Cheap deterministic "signature" line so each thumbnail looks distinct
  // without needing a real uploaded image.
  const d =
    seed % 2 === 0
      ? 'M4 30 C 10 10, 18 40, 26 20 S 42 8, 50 26 S 66 38, 74 18 S 90 12, 96 24'
      : 'M4 22 C 14 34, 20 6, 30 24 S 46 40, 54 16 S 70 6, 80 28 S 92 34, 98 20'

  return (
    <svg viewBox="0 0 100 44" className="h-full w-full">
      <path d={d} fill="none" stroke="#131A2C" strokeWidth="2" strokeLinecap="round" />
    </svg>
  )
}