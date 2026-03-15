FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

# Install build tools
RUN go install github.com/a-h/templ/cmd/templ@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate code
RUN templ generate
RUN sqlc generate

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o specto ./cmd/web

# ---

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /app/specto .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/static ./static

EXPOSE 3000
CMD ["./specto"]
