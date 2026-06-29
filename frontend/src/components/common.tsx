import {useState, type ReactNode} from 'react';
import {ChevronDown, ChevronRight} from 'lucide-react';

export function CollapsibleSection({label, children, defaultOpen = false}: {label: string; children: ReactNode; defaultOpen?: boolean}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className="border-l-2 border-slate-200 pl-3">
      <button type="button" className="flex items-center gap-1 text-xs font-semibold text-slate-700 hover:text-slate-900" onClick={() => setOpen(!open)}>
        {open ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        {label}
      </button>
      {open && <div className="mt-2">{children}</div>}
    </div>
  );
}

export function MiniMetric({label, value}: {label: string; value: ReactNode}) {
  return (
    <div className="rounded-lg border border-slate-100 bg-slate-50 px-3 py-2">
      <p className="text-[10px] font-semibold uppercase tracking-wider text-slate-400">{label}</p>
      <p className="mt-0.5 text-sm font-bold text-slate-900">{value}</p>
    </div>
  );
}

export function Panel({icon, title, subtitle, children, compact, summary}: {icon?: ReactNode; title: string; subtitle?: string; children?: ReactNode; compact?: boolean; summary?: string}) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white">
      <div className={`flex items-center gap-2 ${compact ? 'px-4 py-2.5' : 'px-4 py-3'}`}>
        {icon && <span className="text-slate-500">{icon}</span>}
        <span className={`font-semibold text-slate-950 ${compact ? 'text-xs' : 'text-sm'}`}>{title}</span>
        {subtitle && <span className="text-xs text-slate-400">{subtitle}</span>}
      </div>
      {summary && <p className="border-t border-slate-100 px-4 py-2 text-xs text-slate-500">{summary}</p>}
      {children && <div className={`border-t border-slate-100 ${compact ? 'px-4 py-2.5' : 'px-4 py-3'}`}>{children}</div>}
    </div>
  );
}

export function SecondaryButton({label, onClick, icon, disabled, variant}: {label: string; onClick: () => void; icon?: ReactNode; disabled?: boolean; variant?: 'default' | 'blue' | 'green' | 'amber' | 'red'}) {
  const variants = {
    default: 'border-slate-200 text-slate-700 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-950',
    blue: 'border-blue-200 text-blue-700 hover:border-blue-300 hover:bg-blue-50',
    green: 'border-emerald-200 text-emerald-700 hover:border-emerald-300 hover:bg-emerald-50',
    amber: 'border-amber-200 text-amber-700 hover:border-amber-300 hover:bg-amber-50',
    red: 'border-red-200 text-red-700 hover:border-red-300 hover:bg-red-50',
  };
  return (
    <button type="button" onClick={onClick} disabled={disabled} className={`inline-flex items-center justify-center gap-1.5 rounded-xl border bg-white px-3.5 py-2 text-xs font-bold shadow-sm shadow-slate-200/60 transition hover:-translate-y-0.5 hover:shadow-md active:translate-y-0 active:shadow-sm disabled:pointer-events-none disabled:opacity-40 ${variants[variant ?? 'default']}`}>
      {icon}{label}
    </button>
  );
}

export function PipelineButton({label, onClick, tone, disabled}: {label: string; onClick: () => void; tone?: 'primary'; disabled?: boolean}) {
  const classes = tone === 'primary'
    ? 'border-slate-950 bg-slate-950 text-white shadow-slate-300/80 hover:bg-slate-800 hover:shadow-md'
    : 'border-slate-200 bg-white text-slate-800 shadow-slate-200/70 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-950 hover:shadow-md';
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`inline-flex min-w-20 items-center justify-center rounded-xl border px-4 py-2 text-xs font-bold shadow-sm transition-all hover:-translate-y-0.5 active:translate-y-0 active:shadow-sm disabled:pointer-events-none disabled:opacity-40 ${classes}`}
    >
      {label}
    </button>
  );
}

export function IconButton({label, onClick, children, submit, full, disabled}: {label: string; onClick?: () => void; children?: ReactNode; submit?: boolean; full?: boolean; disabled?: boolean}) {
  return (
    <button onClick={onClick} disabled={disabled} type={submit ? 'submit' : 'button'}
      className={`inline-flex items-center justify-center gap-1.5 rounded-xl border border-slate-950 bg-slate-950 px-3.5 py-2 text-xs font-bold text-white shadow-sm shadow-slate-300/80 transition-all hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-md active:translate-y-0 active:shadow-sm disabled:pointer-events-none disabled:opacity-40 ${full ? 'flex-1 justify-center' : ''}`}>
      {children}{label}
    </button>
  );
}

export function StatusBadge({text, color}: {text: string; color?: 'slate' | 'green' | 'blue' | 'amber' | 'red'}) {
  const colors = {
    slate: 'bg-slate-100 text-slate-600',
    green: 'bg-green-100 text-green-700',
    blue: 'bg-blue-100 text-blue-700',
    amber: 'bg-amber-100 text-amber-700',
    red: 'bg-red-100 text-red-700',
  };
  return <span className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${colors[color ?? 'slate']}`}>{text}</span>;
}

export function TextInput({label, value, onChange}: {label: string; value: string; onChange: (v: string) => void}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-slate-500">{label}</label>
      <input type="text" value={value} onChange={(e) => onChange(e.target.value)} className="w-full rounded-md border border-slate-200 px-3 py-1.5 text-sm text-slate-950 focus:border-slate-400 focus:outline-none" />
    </div>
  );
}

export function TextArea({label, value, onChange, rows}: {label: string; value: string; onChange: (v: string) => void; rows?: number}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-slate-500">{label}</label>
      <textarea value={value} onChange={(e) => onChange(e.target.value)} rows={rows || 4} className="w-full rounded-md border border-slate-200 px-3 py-2 text-sm text-slate-950 focus:border-slate-400 focus:outline-none" />
    </div>
  );
}

export function SelectInput({label, value, onChange, options}: {label: string; value: string; onChange: (v: string) => void; options: [string, string][]}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-slate-500">{label}</label>
      <select value={value} onChange={(e) => onChange(e.target.value)} className="w-full rounded-md border border-slate-200 px-3 py-1.5 text-sm text-slate-950 focus:border-slate-400 focus:outline-none">
        {options.map(([k, l]) => <option key={k} value={k}>{l}</option>)}
      </select>
    </div>
  );
}

export function CheckInput({checked, label, onChange}: {checked: boolean; label: string; onChange: (v: boolean) => void}) {
  return (
    <label className="flex items-center gap-2 text-xs text-slate-700">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} className="rounded border-slate-300" />
      {label}
    </label>
  );
}

export function EmptyState({text}: {text: string}) {
  return <div className="rounded-lg border border-dashed border-slate-200 py-8 text-center text-xs text-slate-400">{text}</div>;
}
