# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ghapp .

# Final stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /ghapp /usr/local/bin/ghapp

WORKDIR /app

EXPOSE 8080

ENTRYPOINT ["ghapp"]
