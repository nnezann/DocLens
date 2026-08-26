// Right-hand illustrated panel used across all auth screens.
// Recreates the repeating hand-drawn documents / books / magnifier
// motif from the mockups as a tiled SVG pattern so it scales cleanly
// at any viewport height without shipping a raster asset.

const Book = ({ x, y, rotate = 0 }) => (
  <g transform={`translate(${x} ${y}) rotate(${rotate})`} stroke="#CBD2DE" strokeWidth="1.3" fill="none">
    <path d="M0 10 L28 0 L60 10 L32 22 Z" />
    <path d="M0 10 L0 34 L32 46 L32 22 Z" />
    <path d="M32 22 L32 46 L60 34 L60 10 Z" strokeDasharray="2 3" />
    <rect x="10" y="26" width="16" height="10" rx="1.5" />
    <path d="M10 32 h16" />
  </g>
)

const Magnifier = ({ x, y, rotate = 0 }) => (
  <g transform={`translate(${x} ${y}) rotate(${rotate})`} stroke="#CBD2DE" strokeWidth="1.3" fill="none">
    <circle cx="14" cy="14" r="12" />
    <line x1="22" y1="22" x2="34" y2="34" strokeLinecap="round" />
    <circle cx="14" cy="14" r="5" strokeDasharray="1.5 2.5" />
  </g>
)

const Document = ({ x, y, rotate = 0 }) => (
  <g transform={`translate(${x} ${y}) rotate(${rotate})`} stroke="#CBD2DE" strokeWidth="1.3" fill="none">
    <path d="M2 0 L26 0 L34 8 L34 46 L2 46 Z" strokeDasharray="2 3" />
    <path d="M26 0 L26 8 L34 8" />
    <path d="M8 16 h18" />
    <path d="M8 23 h18" />
    <path d="M8 30 h12" />
  </g>
)

const Tag = ({ x, y, rotate = 0 }) => (
  <g transform={`translate(${x} ${y}) rotate(${rotate})`} stroke="#CBD2DE" strokeWidth="1.3" fill="none">
    <path d="M2 16 L2 4 Q2 0 6 0 L28 0 Q32 0 32 4 L32 30 Q32 34 28 34 L10 40 Z" strokeDasharray="2 3" />
    <circle cx="17" cy="12" r="4" />
    <path d="M8 22 h16" />
    <path d="M8 28 h16" />
  </g>
)

const TILE = [
  { C: Document, x: 4, y: 0, rotate: -8 },
  { C: Magnifier, x: 92, y: 18, rotate: 0 },
  { C: Book, x: 168, y: 4, rotate: -4 },
  { C: Tag, x: 40, y: 96, rotate: 6 },
  { C: Book, x: 130, y: 108, rotate: 4 },
  { C: Magnifier, x: 8, y: 168, rotate: -10 },
  { C: Document, x: 150, y: 176, rotate: 8 },
]

export default function PatternBackground() {
  return (
    <div className="relative h-full w-full overflow-hidden bg-white">
      <svg className="h-full w-full" preserveAspectRatio="xMidYMid slice">
        <defs>
          <pattern id="doclens-doodles" width="220" height="260" patternUnits="userSpaceOnUse" patternTransform="rotate(0)">
            {TILE.map(({ C, x, y, rotate }, i) => (
              <C key={i} x={x} y={y} rotate={rotate} />
            ))}
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#doclens-doodles)" />
      </svg>
      {/* soft fade at the seam so it reads as an edge, not a hard crop */}
      <div className="pointer-events-none absolute inset-y-0 left-0 w-24 bg-gradient-to-r from-white to-transparent" />
    </div>
  )
}
