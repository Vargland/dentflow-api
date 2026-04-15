FROM golang:1.24-alpine AS builder
WORKDIR /app

# Allow building with a newer module directive
ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/api .
EXPOSE 8080
CMD ["./api"]
