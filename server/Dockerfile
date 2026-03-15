# Othrys Server — Production Docker Image
# Build: docker build -t othrys-server -f server/Dockerfile .
# Run:   docker run -p 8080:8080 --env-file .env othrys-server

FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /build/othrys-server ./cmd/server

# Minimal runtime — no git needed (git ops are client-side only)
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/othrys-server ./
COPY migrations/ ./migrations/

EXPOSE 8080

ENTRYPOINT ["./othrys-server"]
