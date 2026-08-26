import {
  Bell,
  Building2,
  Calendar,
  ChevronDown,
  Folder,
  Home,
  LogOut,
  Search,
  Settings,
  Tag,
  UserCircle2,
} from 'lucide-react'

const NAV_ITEMS = [
  { label: 'Home', icon: Home, active: true },
  { label: 'Cases', icon: Folder, active: false },
  { label: 'Requests', icon: Tag, active: false },
  { label: 'Reports', icon: Calendar, active: false },
]

function NavItem({ label, icon: Icon, active }) {
  return (
    <button
      type="button"
      className={`flex w-full items-center gap-3 rounded-xl px-4 py-3 text-[15px] font-medium transition ${
        active ? 'bg-slate-100 text-ink-900' : 'text-slate-500 hover:bg-slate-50 hover:text-ink-900'
      }`}
    >
      <Icon size={19} strokeWidth={1.8} />
      {label}
    </button>
  )
}

export default function DashboardLayout({ children }) {
  return (
    <div className="flex min-h-screen bg-white">
      {/* Sidebar */}
      <aside className="flex w-64 shrink-0 flex-col justify-between border-r border-slate-100 px-5 py-6">
        <div>
          <div className="mb-8 flex items-center gap-2 px-2">
            <svg width="26" height="26" viewBox="0 0 26 26" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M6 20 L18 5"
                stroke="#131A2C"
                strokeWidth="2.4"
                strokeLinecap="round"
              />
              <circle cx="7" cy="19" r="4.2" stroke="#131A2C" strokeWidth="2" fill="white" />
            </svg>
            <span className="font-display text-xl font-bold text-ink-900">DocLens</span>
          </div>

          <nav className="space-y-1">
            {NAV_ITEMS.map((item) => (
              <NavItem key={item.label} {...item} />
            ))}
          </nav>
        </div>

        <div className="space-y-1">
          <NavItem label="Settings" icon={Settings} />
          <button
            type="button"
            className="flex w-full items-center gap-3 rounded-xl px-4 py-3 text-[15px] font-medium text-slate-500 hover:bg-slate-50 hover:text-ink-900"
          >
            <LogOut size={19} strokeWidth={1.8} />
            Logout
          </button>
        </div>
      </aside>

      {/* Main column */}
      <div className="flex flex-1 flex-col">
        {/* Topbar */}
        <header className="flex items-center gap-4 border-b border-slate-100 px-8 py-4">
          <button
            type="button"
            className="flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm font-medium text-ink-900"
          >
            <Building2 size={16} strokeWidth={1.8} />
            Seraphin &amp; co.Ltd
            <ChevronDown size={16} strokeWidth={1.8} className="text-slate-400" />
          </button>

          <div className="relative flex-1 max-w-md">
            <Search size={17} className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="search"
              placeholder="Search investigations, documents..."
              className="w-full rounded-full border border-slate-200 bg-white py-2.5 pl-11 pr-4 text-sm text-ink-900 placeholder:text-slate-400 focus:border-ink-900 focus:outline-none focus:ring-1 focus:ring-ink-900"
            />
          </div>

          <div className="ml-auto flex items-center gap-5">
            <button type="button" aria-label="Notifications" className="text-ink-900">
              <Bell size={20} strokeWidth={1.8} />
            </button>
            <div className="flex items-center gap-2">
              <UserCircle2 size={30} strokeWidth={1.5} className="text-ink-900" />
              <div className="leading-tight">
                <p className="text-sm font-semibold text-ink-900">Mark Ferdinand</p>
                <p className="text-xs text-slate-400">Employee</p>
              </div>
            </div>
          </div>
        </header>

        <main className="flex-1 px-8 py-8">{children}</main>
      </div>
    </div>
  )
}
