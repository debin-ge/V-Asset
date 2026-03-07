# AGENTS.md
Agent guide for the V-Asset monorepo.

## Scope
- Repository root: `/Users/gedebin/Documents/Code/V-Asset`
- Frontend: Next.js 16 + React 19 + TypeScript + TailwindCSS 4
- Backend: Go microservices (`api-gateway`, `auth-service`, `media-service`, `asset-service`)
- Services communicate by gRPC; API Gateway exposes HTTP/WebSocket

## Cursor/Copilot Rules
- `.cursorrules`: not present
- `.cursor/rules/`: not present
- `.github/copilot-instructions.md`: not present
- No external agent rule files currently override this file

## Repository Map
- `frontend-service/`: UI and web client
- `api-gateway/`: HTTP routes, middleware, gRPC fan-out
- `auth-service/`: user auth/session/token logic
- `media-service/`: parse/download orchestration
- `asset-service/`: history/quota/proxy/cookie management
- `docker-compose.yml`: local full-stack runtime

## Build / Lint / Test Commands
Run from each service directory unless noted.

### Frontend (`frontend-service`)
```bash
npm install
npm run dev
npm run build
npm run start
npm run lint
```
Focused lint:
```bash
npm run lint -- app/page.tsx
npm run lint -- components/home
```
Frontend testing:
- No test script currently exists in `frontend-service/package.json`
- If tests are added, add `npm run test` and document single-test usage

### Go Services
Common Make targets in all Go services:
```bash
make deps
make proto
make build
make run
make test
make clean
make docker-build
make docker-up
make docker-down
```
Default test command used by `make test`:
```bash
go test -v ./...
```
Single-test commands (important):
```bash
# one test function
go test -v ./internal/service -run '^TestAuthService_Login$'

# one subtest
go test -v ./internal/service -run 'TestParserService_ParseURL/cache hit'

# rerun without test cache
go test -v ./internal/handler -run '^TestParseURL$' -count=1
```

## Linting and Static Analysis
No repo-level `golangci-lint` config is committed.
Go baseline checks:
```bash
gofmt -w ./...
go vet ./...
go test -v ./...
```
Frontend baseline checks:
```bash
npm run lint
npm run build
```

## TypeScript / React Style Guidelines
### Imports
- Prefer alias imports via `@/*` (configured in `frontend-service/tsconfig.json`)
- Import order: framework/external -> internal alias -> relative
- Use `import type` for type-only imports
- `lib/*` contains relative imports; keep local style consistent when editing

### Formatting
- Preserve surrounding file style and avoid unrelated reformatting
- Semicolon usage is mixed; follow the local file convention

### Types
- TypeScript strict mode is enabled; avoid `any`
- Define explicit interfaces/types for API contracts and payloads
- Use narrow unions for state/status values

### Naming
- Components: PascalCase (`ResultCard`, `AuthModal`)
- Hooks: `useXxx` (`useDownload`, `useAuth`)
- Variables/functions: camelCase
- Constants: UPPER_SNAKE_CASE only for true constants

### Error Handling
- Wrap async UI actions in `try/catch`
- Show user-facing errors with `toast.error(...)`
- Normalize unknown errors safely (`error instanceof Error ? ... : ...`)

### API and State Patterns
- Central axios auth/response handling is in `frontend-service/lib/api-client.ts`
- Keep endpoint wrappers in `frontend-service/lib/api/*` with strong types
- For websocket flows, always unsubscribe during cleanup

## Go Style Guidelines
### Architecture
- Keep boundaries clear: `handler` -> `service` -> `repository`
- Keep `cmd/main.go` focused on wiring/bootstrap

### Formatting and Imports
- Always run `gofmt` on edited Go files
- Keep imports gofmt/goimports-compatible

### Types and Naming
- Prefer concrete structs; add interfaces only when justified
- Exported identifiers: PascalCase
- Unexported identifiers: camelCase
- Reusable package error vars: `ErrXxx`

### Error Handling
- Return errors instead of panicking in request paths
- Wrap lower-level failures with context using `%w`
- Validate input early and return fast

### Logging, Context, Concurrency
- Use structured logs when logger supports it (zap in media-service)
- Never log secrets/tokens/passwords
- Propagate `context.Context` through DB/Redis/gRPC/HTTP boundaries

## Change Discipline for Agents
- Keep changes focused; avoid unrelated refactors
- Do not manually edit generated protobuf files (`*.pb.go`, `*_grpc.pb.go`)
- If `.proto` changes, run `make proto` in affected services
- Preserve API compatibility unless the task explicitly requires breaking changes

## Common Ports
- Frontend: `3000`
- API Gateway: `8080`
- Auth gRPC: `9001`
- Media gRPC: `9002`
- Asset gRPC: `9004`
