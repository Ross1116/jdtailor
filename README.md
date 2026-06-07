# JD Tailor

Local-first job application workflow tool.

## Phase 1 Bootstrap

This slice provides the Wails shell, React/Vite/Tailwind console, Go backend
health checks, repo-local SQLite storage, local logs, and an LLM settings stub.
Real LLM calls and PDF rendering are deferred.

## Requirements

- Go 1.24+
- Node.js 20+
- npm
- Wails v2

Install Wails if needed:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Verify toolchain:

```powershell
go version
node --version
npm --version
wails doctor
```

If `wails` is not on `PATH`, run it from:

```powershell
$env:USERPROFILE\go\bin\wails.exe
```

## Run

```powershell
cd frontend
npm install
cd ..
wails dev
```

## Test

```powershell
go test ./...
cd frontend
npm run build
```

Local runtime files are ignored by Git:

- `data/app.db`
- `logs/app.log`
- `generated/`
