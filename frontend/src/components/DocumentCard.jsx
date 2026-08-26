import { Eye, Download, RefreshCcw, FileText, Image as ImageIcon, Folder } from 'lucide-react'

const KIND_ICON = { PDF: FileText, PNG: ImageIcon, JPG: ImageIcon }

export default function DocumentCard({ name, kind, size, date, thumbnail }) {
  const KindIcon = KIND_ICON[kind] ?? Folder

  return (
    <div className="rounded-2xl border border-slate-100 p-4">
      <div className="flex items-start gap-3">
        <div className="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-slate-50">
          {thumbnail ?? <KindIcon size={22} strokeWidth={1.5} className="text-slate-400" />}
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-1.5">
            <p className="truncate font-medium text-ink-900">{name}</p>
            <KindIcon size={14} className="shrink-0 text-slate-400" />
          </div>
          <p className="mt-1 text-sm text-slate-400">
            {kind} &bull; {size}
          </p>
          <p className="mt-0.5 text-sm text-slate-400">{date}</p>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          className="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm font-medium text-ink-900 hover:bg-slate-50"
        >
          <Eye size={15} /> Preview
        </button>
        <button
          type="button"
          className="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm font-medium text-ink-900 hover:bg-slate-50"
        >
          <Download size={15} /> Download
        </button>
        <button
          type="button"
          className="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm font-medium text-ink-900 hover:bg-slate-50"
        >
          <RefreshCcw size={15} /> Replace
        </button>
      </div>
    </div>
  )
}