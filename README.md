# JD Tailor

Local-first job application workflow tool.

## Phase Status

- Phase 1 - Bootstrap: complete
- Phase 2 - Candidate Context Builder: complete
- Phase 3 - Resume Core: in progress

## Phase 1 Bootstrap

This slice provides the Wails shell, React/Vite/Tailwind console, Go backend
health checks, repo-local SQLite storage, local logs, and an LLM settings stub.
Real LLM calls and PDF rendering are deferred.

## Phase 2 Candidate Context Builder

This slice builds the local truth layer used by resume generation:

- Candidate source import from pasted text, TXT, Markdown, and LaTeX.
- Section detection with editable headings and section types.
- Evidence fact extraction with quote-backed atoms, origin metadata, duplicate checks, and review status.
- Fact review queue for approve/reject/edit flows.
- Candidate claim generation from approved facts.
- Claim verification against source facts, blocked contexts, unsupported tools, unsupported metrics, and title/seniority rules.
- Claim review queue with approved, restricted, blocked, and rejected states.
- Blocked-claim registry seeded with default guardrails and editable in the UI.
- Candidate profile and mental-model records for locked identity, employment, education, projects, allowed aliases, and blocked aliases.
- Prompt rules registry for reusable truth, style, validation, and resume-writing constraints.

Phase 2 intentionally stops before full resume assembly. Job parsing, match maps,
fit analysis, bullet diagnostics, and bullet selection have begun as Phase 3
foundation work, but final resume JSON/PDF assembly is still Phase 3.

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
