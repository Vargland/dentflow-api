BINARY=bin/api
CMD=./cmd/api

.PHONY: run build test fmt vet lint migrate-up migrate-down sqlc pre-commit

## run: start the API in development mode
run:
	go run $(CMD)

## build: compile the binary
build:
	go build -o $(BINARY) $(CMD)

## test: run all tests with race detector
test:
	go test ./... -race -count=1

## fmt: format all Go files
fmt:
	go fmt ./...

## vet: run go vet
vet:
	go fmt ./... && go vet ./...

## lint: run staticcheck
lint:
	staticcheck ./...

## tidy: tidy go.mod and go.sum
tidy:
	go mod tidy

## sqlc: regenerate sqlc types from SQL queries
sqlc:
	sqlc generate

## migrate-up: apply all pending migrations using golang-migrate
migrate-up:
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path internal/db/migrations \
		-database "$(DATABASE_URL)" up

## migrate-down: rollback the last migration
migrate-down:
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path internal/db/migrations \
		-database "$(DATABASE_URL)" down 1

## pre-commit: run all checks (used by git hook)
pre-commit: fmt vet test
