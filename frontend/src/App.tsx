import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  Database,
  FileText,
  Folder,
  RefreshCcw,
  Save,
  Settings as SettingsIcon,
} from 'lucide-react';
import {
  GetHealth,
  GetRecentEvents,
  GetSettings,
  SaveSettings,
} from '../wailsjs/go/main/App';

type Health = {
  version: string;
  storage_status: string;
  db_path: string;
  log_path: string;
  generated_path: string;
  pdf_renderer: string;
};

type Settings = {
  provider: string;
  model: string;
  api_key_configured: boolean;
};

type AppEvent = {
  id: number;
  level: string;
  message: string;
  created_at: string;
};

type LoadState = 'loading' | 'ready' | 'error';

function App() {
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [error, setError] = useState('');
  const [health, setHealth] = useState<Health | null>(null);
  const [settings, setSettings] = useState<Settings>({
    provider: 'openai',
    model: '',
    api_key_configured: false,
  });
  const [events, setEvents] = useState<AppEvent[]>([]);
  const [saving, setSaving] = useState(false);

  const statusText = useMemo(() => {
    if (loadState === 'loading') {
      return 'Starting';
    }
    if (loadState === 'error') {
      return 'Needs attention';
    }
    return 'Ready';
  }, [loadState]);

  async function load() {
    setLoadState('loading');
    setError('');
    try {
      const [nextHealth, nextSettings, nextEvents] = await Promise.all([
        GetHealth(),
        GetSettings(),
        GetRecentEvents(),
      ]);
      setHealth(nextHealth as Health);
      setSettings(nextSettings as Settings);
      setEvents((nextEvents ?? []) as AppEvent[]);
      setLoadState('ready');
    } catch (err) {
      setLoadState('error');
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function saveSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError('');
    try {
      const nextSettings = await SaveSettings({
        provider: settings.provider,
        model: settings.model,
      });
      setSettings(nextSettings as Settings);
      const nextEvents = await GetRecentEvents();
      setEvents((nextEvents ?? []) as AppEvent[]);
      setLoadState('ready');
    } catch (err) {
      setLoadState('error');
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  return (
    <main className="min-h-screen bg-[#f6f8fb]">
      <section className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-500">
              JD Tailor
            </p>
            <h1 className="mt-1 text-2xl font-semibold text-slate-950">
              Local foundation console
            </h1>
          </div>
          <div className="flex items-center gap-3">
            <StatusPill state={loadState} text={statusText} />
            <button
              type="button"
              onClick={load}
              className="inline-flex h-10 items-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50"
            >
              <RefreshCcw size={16} />
              Refresh
            </button>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-4 px-5 py-5 lg:grid-cols-[1.2fr_0.8fr]">
        <div className="space-y-4">
          {error && (
            <div className="flex items-start gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">
              <AlertCircle className="mt-0.5 shrink-0" size={18} />
              <span>{error}</span>
            </div>
          )}

          <Panel
            icon={<Database size={18} />}
            title="Storage"
            subtitle="Repo-local SQLite, logs, and generated output paths."
          >
            <div className="grid gap-3 md:grid-cols-2">
              <PathRow label="Database" value={health?.db_path} icon={<Database size={16} />} />
              <PathRow label="Log file" value={health?.log_path} icon={<FileText size={16} />} />
              <PathRow label="Generated" value={health?.generated_path} icon={<Folder size={16} />} />
              <MetricRow label="PDF renderer" value={health?.pdf_renderer ?? 'not_configured'} />
            </div>
          </Panel>

          <Panel
            icon={<Activity size={18} />}
            title="Recent events"
            subtitle="Important backend events mirrored into SQLite."
          >
            <div className="overflow-hidden rounded-md border border-slate-200">
              {events.length === 0 ? (
                <p className="px-4 py-5 text-sm text-slate-500">No events yet.</p>
              ) : (
                <ul className="divide-y divide-slate-200">
                  {events.map((event) => (
                    <li key={event.id} className="grid gap-2 px-4 py-3 sm:grid-cols-[88px_1fr_180px]">
                      <span className="text-xs font-semibold uppercase text-slate-500">
                        {event.level}
                      </span>
                      <span className="text-sm text-slate-900">{event.message}</span>
                      <time className="text-xs text-slate-500">{formatDate(event.created_at)}</time>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </Panel>
        </div>

        <div className="space-y-4">
          <Panel
            icon={<SettingsIcon size={18} />}
            title="LLM settings"
            subtitle="Provider stub only. No network call in this slice."
          >
            <form className="space-y-4" onSubmit={saveSettings}>
              <label className="block">
                <span className="text-sm font-medium text-slate-700">Provider</span>
                <select
                  value={settings.provider}
                  onChange={(event) => setSettings({...settings, provider: event.target.value})}
                  className="mt-1 h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
                >
                  <option value="openai">OpenAI</option>
                </select>
              </label>

              <label className="block">
                <span className="text-sm font-medium text-slate-700">Model</span>
                <input
                  value={settings.model}
                  onChange={(event) => setSettings({...settings, model: event.target.value})}
                  placeholder="Leave blank for now"
                  className="mt-1 h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
                />
              </label>

              <MetricRow
                label="API key"
                value={settings.api_key_configured ? 'configured' : 'not_configured'}
              />

              <button
                type="submit"
                disabled={saving}
                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-slate-950 px-3 text-sm font-medium text-white shadow-sm hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400"
              >
                <Save size={16} />
                {saving ? 'Saving' : 'Save settings'}
              </button>
            </form>
          </Panel>

          <Panel
            icon={<CheckCircle2 size={18} />}
            title="Foundation state"
            subtitle="First slice gates before truth workflow screens."
          >
            <dl className="space-y-3">
              <MetricRow label="App version" value={health?.version ?? 'unknown'} />
              <MetricRow label="Storage" value={health?.storage_status ?? 'checking'} />
              <MetricRow label="LLM calls" value="deferred" />
              <MetricRow label="Resume/PDF" value="deferred" />
            </dl>
          </Panel>
        </div>
      </section>
    </main>
  );
}

function StatusPill({state, text}: {state: LoadState; text: string}) {
  const styles = {
    loading: 'border-amber-200 bg-amber-50 text-amber-800',
    ready: 'border-emerald-200 bg-emerald-50 text-emerald-800',
    error: 'border-red-200 bg-red-50 text-red-800',
  }[state];

  return (
    <span className={`inline-flex h-10 items-center gap-2 rounded-md border px-3 text-sm font-medium ${styles}`}>
      {state === 'ready' ? <CheckCircle2 size={16} /> : <AlertCircle size={16} />}
      {text}
    </span>
  );
}

function Panel({
  icon,
  title,
  subtitle,
  children,
}: {
  icon: JSX.Element;
  title: string;
  subtitle: string;
  children: JSX.Element;
}) {
  return (
    <section className="rounded-md border border-slate-200 bg-white p-4 shadow-sm">
      <div className="mb-4 flex items-start gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-700">
          {icon}
        </div>
        <div>
          <h2 className="text-base font-semibold text-slate-950">{title}</h2>
          <p className="mt-1 text-sm text-slate-500">{subtitle}</p>
        </div>
      </div>
      {children}
    </section>
  );
}

function PathRow({label, value, icon}: {label: string; value?: string; icon: JSX.Element}) {
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
      <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase text-slate-500">
        {icon}
        {label}
      </div>
      <p className="break-all text-sm text-slate-900">{value ?? 'checking'}</p>
    </div>
  );
}

function MetricRow({label, value}: {label: string; value: string}) {
  return (
    <div className="flex min-h-10 items-center justify-between gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-2">
      <dt className="text-sm font-medium text-slate-600">{label}</dt>
      <dd className="text-right text-sm font-semibold text-slate-950">{value}</dd>
    </div>
  );
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

export default App;
