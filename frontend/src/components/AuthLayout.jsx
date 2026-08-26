import PatternBackground from './PatternBackground'

export default function AuthLayout({ children }) {
  return (
    <div className="flex min-h-screen w-full bg-white">
      <div className="flex w-full items-center justify-center px-6 py-16 sm:px-12 lg:w-1/2 lg:px-20">
        <div className="w-full max-w-sm">{children}</div>
      </div>
      <div className="hidden lg:block lg:w-1/2">
        <PatternBackground />
      </div>
    </div>
  )
}
