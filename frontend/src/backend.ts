import {
  GetHealth as WailsGetHealth,
  GetRecentEvents as WailsGetRecentEvents,
  GetSettings as WailsGetSettings,
  GetToolStatus as WailsGetToolStatus,
  InstallTectonic as WailsInstallTectonic,
  RenderSamplePDF as WailsRenderSamplePDF,
  SaveAPIKey as WailsSaveAPIKey,
  SaveSettings as WailsSaveSettings,
  TestLLM as WailsTestLLM,
} from '../wailsjs/go/main/App';

const hasWailsBackend = () => Boolean(window.go?.main?.App);

const mockSettings = {
  provider: 'openrouter',
  model: 'deepseek/deepseek-v4-flash',
  api_key_configured: false,
};

const mockEvents = [
  {
    id: 1,
    level: 'info',
    message: 'frontend mock backend active',
    created_at: new Date().toISOString(),
  },
];

export async function GetHealth() {
  if (hasWailsBackend()) {
    return WailsGetHealth();
  }
  return {
    version: 'frontend-dev',
    storage_status: 'mock',
    db_path: 'mock://data/app.db',
    log_path: 'mock://logs/app.log',
    generated_path: 'mock://generated',
    pdf_renderer: 'mock',
  };
}

export async function GetSettings() {
  if (hasWailsBackend()) {
    return WailsGetSettings();
  }
  return mockSettings;
}

export async function SaveSettings(input: {provider: string; model: string}) {
  if (hasWailsBackend()) {
    return WailsSaveSettings(input);
  }
  mockSettings.provider = input.provider;
  mockSettings.model = input.model || modelDefault(input.provider);
  mockEvents.unshift(mockEvent('info', 'mock settings saved'));
  return mockSettings;
}

export async function GetToolStatus() {
  if (hasWailsBackend()) {
    return WailsGetToolStatus();
  }
  return {
    api_key_configured: mockSettings.api_key_configured,
    api_key_source: mockSettings.api_key_configured ? 'mock' : '',
    env_local_path: 'mock://.env.local',
    tectonic_status: 'mock',
    tectonic_path: 'mock://tools/tectonic/tectonic.exe',
    generated_path: 'mock://generated',
  };
}

export async function SaveAPIKey(input: {api_key: string; provider: string}) {
  if (hasWailsBackend()) {
    return WailsSaveAPIKey(input);
  }
  mockSettings.provider = input.provider;
  mockSettings.api_key_configured = input.api_key.trim() !== '';
  mockEvents.unshift(mockEvent('info', 'mock API key saved'));
  return GetToolStatus();
}

export async function TestLLM() {
  if (hasWailsBackend()) {
    return WailsTestLLM();
  }
  const success = mockSettings.api_key_configured;
  mockEvents.unshift(mockEvent(success ? 'info' : 'error', success ? 'mock LLM smoke test succeeded' : 'mock LLM smoke test failed: missing API key'));
  return {
    success,
    provider: mockSettings.provider,
    model: mockSettings.model,
    text: success ? 'JD Tailor LLM check' : '',
    latency_ms: 12,
    status_code: success ? 200 : 0,
    error: success ? '' : 'OPENROUTER_API_KEY is missing',
  };
}

export async function InstallTectonic() {
  if (hasWailsBackend()) {
    return WailsInstallTectonic();
  }
  mockEvents.unshift(mockEvent('info', 'mock Tectonic installed'));
  return {
    success: true,
    status: 'mock',
    executable_path: 'mock://tools/tectonic/tectonic.exe',
    error: '',
  };
}

export async function RenderSamplePDF() {
  if (hasWailsBackend()) {
    return WailsRenderSamplePDF();
  }
  mockEvents.unshift(mockEvent('info', 'mock sample PDF rendered'));
  return {
    success: true,
    tex_path: 'mock://generated/sample-pdf/sample.tex',
    pdf_path: 'mock://generated/sample-pdf/sample.pdf',
    error: '',
  };
}

export async function GetRecentEvents() {
  if (hasWailsBackend()) {
    return WailsGetRecentEvents();
  }
  return mockEvents;
}

function mockEvent(level: string, message: string) {
  return {
    id: Date.now(),
    level,
    message,
    created_at: new Date().toISOString(),
  };
}

function modelDefault(provider: string) {
  return provider === 'openai' ? 'gpt-5.4-mini' : 'deepseek/deepseek-v4-flash';
}
