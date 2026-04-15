# DentFlow API — Setup Guide

## 1. Install Go 1.22+

Download and install from https://go.dev/dl/  
Choose **go1.26.2.windows-amd64.msi** and run the installer.

Verify: open a new terminal and run:
```
go version
```

## 2. Install tools

```bash
# golang-migrate CLI (for running migrations)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# sqlc (only needed if you change SQL queries)
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# staticcheck (linter)
go install honnef.co/go/tools/cmd/staticcheck@latest
```

## 3. Download dependencies

```bash
cd E:/Work/odonto/dentflow-api
go mod tidy
```

## 4. Run migrations against Neon

```bash
# Windows PowerShell (migrate.exe installed at C:\Users\<user>\go\bin\)
$DB="postgresql://neondb_owner:...@.../neondb?sslmode=require"
migrate -path E:/Work/odonto/dentflow-api/internal/db/migrations -database $DB up
```

Or on Git Bash:
```bash
export DATABASE_URL="postgresql://..."
make migrate-up
```

## 5. Start the API

```bash
make run
# or: go run ./cmd/api
```

API will be available at http://localhost:8080

## 6. Test health endpoint

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## Route overview

```
GET    /health
GET    /api/v1/patients              ?q=search
POST   /api/v1/patients
GET    /api/v1/patients/:id
PUT    /api/v1/patients/:id
DELETE /api/v1/patients/:id
GET    /api/v1/patients/:id/odontogram
PUT    /api/v1/patients/:id/odontogram
GET    /api/v1/patients/:id/evolutions
POST   /api/v1/patients/:id/evolutions
PUT    /api/v1/patients/:id/evolutions/:eid
DELETE /api/v1/patients/:id/evolutions/:eid
```

All `/api/v1/` routes require: `Authorization: Bearer <jwt>`
