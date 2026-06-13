# Stage 1: Build the statically compiled Go binary with CGO (for go-sqlite3)
FROM golang:1.25.5-alpine AS builder

# Install required build tools for SQLite CGO compilation
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy module manifests and pre-fetch dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy full source code
COPY . .

# Build Go binary statically
ENV CGO_ENABLED=1
RUN go build -ldflags "-extldflags -static" -o server main.go

# Stage 2: Production scratch-like Alpine runner
FROM alpine:3.20

# Add tzdata and root certificates for TLS security
RUN apk add --no-cache tzdata ca-certificates

WORKDIR /app

# Create a writable directory for persistent SQLite database mounts
RUN mkdir -p /app/data

COPY --from=builder /app/server .

# Set default production environment
ENV PORT=8080
ENV DB_PATH=/app/data/gamer_club.db

EXPOSE 8080

CMD ["./server"]
