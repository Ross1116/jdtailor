import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  Cpu,
  Database,
  FileText,
  Folder,
  KeyRound,
  Play,
  RefreshCcw,
  Save,
  Settings as SettingsIcon,
  Wrench,
} from 'lucide-react';
import {
  GetHealth,
  GetRecentEvents,
  GetSettings,
  GetToolStatus,
  InstallTectonic,
  RenderSamplePDF,
  SaveAPIKey,
  SaveSettings,
  TestLLM,
} from './backend';

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

type ToolStatus = {
  api_key_configured: boolean;
  api_key_source: string;
  env_local_path: string;
  tectonic_status: string;
  tectonic_path: string;
  generated_path: string;
};

type LLMTestResult = {
  success: boolean;
  provider: string;
  model: string;
  text: string;
  latency_ms: number;
  status_code: number;
  error: string;
};

type RenderPDFResult = {
  success: boolean;
  tex_path: string;
  pdf_path: string;
  error: string;
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
    provider: 'openrouter',
    model: '',
    api_key_configured: false,
  });
  const [toolStatus, setToolStatus] = useState<ToolStatus | null>(null);
  const [events, setEvents] = useState<AppEvent[]>([]);
  const [apiKey, setAPIKey] = useState('');
  const [llmResult, setLLMResult] = useState<LLMTestResult | null>(null);
  const [pdfResult, setPDFResult] = useState<RenderPDFResult | null>(null);
  const [busyAction, setBusyAction] = useState('');

  const statusText = useMemo(() => {
    if (loadState === 'loading') {
      return 'Starting';
    }
    if (loadState === 'error') {
      return 'Needs attention';
    }
    return 'Ready';
  }, [loadState]);

  async function refreshEvents() {
    const nextEvents = await GetRecentEvents();
    setEvents((nextEvents ?? []) as AppEvent[]);
  }

  async function load() {
    setLoadState('loading');
    setError('');
    try {
      const [nextHealth, nextSettings, nextStatus, nextEvents] = await Promise.all([
        GetHealth(),
        GetSettings(),
        GetToolStatus(),
        GetRecentEvents(),
      ]);
      setHealth(nextHealth as Health);
      setSettings(nextSettings as Settings);
      setToolStatus(nextStatus as ToolStatus);
      setEvents((nextEvents ?? []) as AppEvent[]);
      setLoadState('ready');
    } catch (err) {
      setLoadState('error');
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function saveSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction('save-settings', async () => {
      const nextSettings = await SaveSettings({
        provider: settings.provider,
        model: settings.model,
      });
      setSettings(nextSettings as Settings);
      await load();
    });
  }

  async function saveAPIKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction('save-key', async () => {
      const nextStatus = await SaveAPIKey({api_key: apiKey, provider: settings.provider});
      setToolStatus(nextStatus as ToolStatus);
      setSettings({...settings, api_key_configured: (nextStatus as ToolStatus).api_key_configured});
      setAPIKey('');
      await refreshEvents();
    });
  }

  async function runLLMTest() {
    await runAction('test-llm', async () => {
      const result = (await TestLLM()) as LLMTestResult;
      setLLMResult(result);
      await refreshEvents();
    });
  }

  async function installTectonic() {
    await runAction('install-tectonic', async () => {
      await InstallTectonic();
      await load();
    });
  }

  async function renderSamplePDF() {
    await runAction('render-pdf', async () => {
      const result = (await RenderSamplePDF()) as RenderPDFResult;
      setPDFResult(result);
      await load();
    });
  }

  async function runAction(name: string, action: () => Promise<void>) {
    setBusyAction(name);
    setError('');
    try {
      await action();
      setLoadState('ready');
    } catch (err) {
      setLoadState('error');
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyAction('');
    }
  }

  useEffect(() => {
    load();
  }, []);

  const apiConfigured = toolStatus?.api_key_configured ?? settings.api_key_configured;
  const tectonicStatus = toolStatus?.tectonic_status ?? health?.pdf_renderer ?? 'checking';

  return (
    <main className="min-h-screen bg-[#f6f8fb]">
      <section className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-500">
              JD Tailor
            </p>
            <h1 className="mt-1 text-2xl font-semibold text-slate-950">
              Phase 1 integration console
            </h1>
          </div>
          <div className="flex items-center gap-3">
            <StatusPill state={loadState} text={statusText} />
            <IconButton label="Refresh" onClick={load} disabled={busyAction !== ''}>
              <RefreshCcw size={16} />
            </IconButton>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl gap-4 px-5 py-5 lg:grid-cols-[1.15fr_0.85fr]">
        <div className="space-y-4">
          {error && (
            <div className="flex items-start gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">
              <AlertCircle className="mt-0.5 shrink-0" size={18} />
              <span>{error}</span>
            </div>
          )}

          <Panel icon={<Database size={18} />} title="Storage" subtitle="Repo-local SQLite, logs, generated output, and tool paths.">
            <div className="grid gap-3 md:grid-cols-2">
              <PathRow label="Database" value={health?.db_path} icon={<Database size={16} />} />
              <PathRow label="Log file" value={health?.log_path} icon={<FileText size={16} />} />
              <PathRow label="Generated" value={health?.generated_path} icon={<Folder size={16} />} />
              <PathRow label="Tectonic" value={toolStatus?.tectonic_path} icon={<Cpu size={16} />} />
            </div>
          </Panel>

          <Panel icon={<Activity size={18} />} title="Recent events" subtitle="Backend failures and successful gate checks.">
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
          <Panel icon={<SettingsIcon size={18} />} title="LLM" subtitle="OpenRouter-first smoke call using local key configuration.">
            <div className="space-y-4">
              <form className="space-y-4" onSubmit={saveSettings}>
                <label className="block">
                  <span className="text-sm font-medium text-slate-700">Provider</span>
                  <select
                    value={settings.provider}
                    onChange={(event) => setSettings({...settings, provider: event.target.value})}
                    className="mt-1 h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
                  >
                    <option value="openrouter">OpenRouter</option>
                    <option value="openai">OpenAI direct</option>
                  </select>
                </label>

                <label className="block">
                  <span className="text-sm font-medium text-slate-700">Model</span>
                  <input
                    value={settings.model}
                    onChange={(event) => setSettings({...settings, model: event.target.value})}
                    placeholder={modelPlaceholder(settings.provider)}
                    className="mt-1 h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
                  />
                </label>

                <IconButton label="Save settings" submit full disabled={busyAction === 'save-settings'}>
                  <Save size={16} />
                </IconButton>
              </form>

              <form className="space-y-3 border-t border-slate-200 pt-4" onSubmit={saveAPIKey}>
                <MetricRow label="API key" value={apiConfigured ? `configured${toolStatus?.api_key_source ? ` (${toolStatus.api_key_source})` : ''}` : 'missing'} />
                <label className="block">
                  <span className="text-sm font-medium text-slate-700">Update key</span>
                  <input
                    value={apiKey}
                    onChange={(event) => setAPIKey(event.target.value)}
                    type="password"
                    placeholder={apiKeyPlaceholder(settings.provider)}
                    className="mt-1 h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
                  />
                </label>
                <IconButton label="Save key" submit full disabled={busyAction === 'save-key' || apiKey.trim() === ''}>
                  <KeyRound size={16} />
                </IconButton>
              </form>

              <div className="space-y-3 border-t border-slate-200 pt-4">
                <IconButton label="Test call" onClick={runLLMTest} full disabled={busyAction === 'test-llm'}>
                  <Play size={16} />
                </IconButton>
                <ResultBox
                  ok={llmResult?.success}
                  title={llmResult ? `${llmResult.model} - ${llmResult.latency_ms}ms` : 'Last result'}
                  text={llmResult ? (llmResult.success ? llmResult.text : llmResult.error) : 'No test run yet.'}
                />
              </div>
            </div>
          </Panel>

          <Panel icon={<Wrench size={18} />} title="PDF" subtitle="Repo-local Tectonic install and sample render gate.">
            <div className="space-y-4">
              <MetricRow label="Renderer" value={tectonicStatus} />
              <div className="grid gap-3 sm:grid-cols-2">
                <IconButton label="Install" onClick={installTectonic} disabled={busyAction === 'install-tectonic'}>
                  <Wrench size={16} />
                </IconButton>
                <IconButton label="Render sample" onClick={renderSamplePDF} disabled={busyAction === 'render-pdf'}>
                  <FileText size={16} />
                </IconButton>
              </div>
              <ResultBox
                ok={pdfResult?.success}
                title="Sample PDF"
                text={pdfResult ? (pdfResult.success ? pdfResult.pdf_path : pdfResult.error) : 'No render yet.'}
              />
            </div>
          </Panel>

          <Panel icon={<CheckCircle2 size={18} />} title="Foundation state" subtitle="Phase 1 gates before truth workflow screens.">
            <dl className="space-y-3">
              <MetricRow label="App version" value={health?.version ?? 'unknown'} />
              <MetricRow label="Storage" value={health?.storage_status ?? 'checking'} />
              <MetricRow label="LLM key" value={apiConfigured ? 'configured' : 'missing'} />
              <MetricRow label="PDF renderer" value={tectonicStatus} />
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

function IconButton({
  children,
  disabled,
  full,
  label,
  onClick,
  submit,
}: {
  children: JSX.Element;
  disabled?: boolean;
  full?: boolean;
  label: string;
  onClick?: () => void;
  submit?: boolean;
}) {
  return (
    <button
      type={submit ? 'submit' : 'button'}
      onClick={onClick}
      disabled={disabled}
      className={`inline-flex h-10 items-center justify-center gap-2 rounded-md bg-slate-950 px-3 text-sm font-medium text-white shadow-sm hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400 ${full ? 'w-full' : ''}`}
      title={label}
    >
      {children}
      {label}
    </button>
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

function ResultBox({ok, text, title}: {ok?: boolean; text: string; title: string}) {
  const tone = ok === undefined ? 'border-slate-200 bg-slate-50' : ok ? 'border-emerald-200 bg-emerald-50' : 'border-red-200 bg-red-50';
  return (
    <div className={`rounded-md border px-3 py-3 ${tone}`}>
      <p className="text-xs font-semibold uppercase text-slate-500">{title}</p>
      <p className="mt-1 break-all text-sm text-slate-900">{text}</p>
    </div>
  );
}

function modelPlaceholder(provider: string) {
  return provider === 'openai' ? 'gpt-5.4-mini' : 'deepseek/deepseek-v4-flash';
}

function apiKeyPlaceholder(provider: string) {
  return provider === 'openai' ? 'OPENAI_API_KEY' : 'OPENROUTER_API_KEY';
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

export default App;
