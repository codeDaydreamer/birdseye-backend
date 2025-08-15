# Stage 1: Build
FROM golang:1.23.3-alpine AS build

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server ./cmd/birdseye/main.go

# Stage 2: Run (Debian-based)
FROM python:3.11-slim

# Install system dependencies required by WeasyPrint and certificates/timezone
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libcairo2-dev \
    libpango1.0-dev \
    libgdk-pixbuf-xlib-2.0-dev \
    libffi-dev \
    tzdata \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*


# Install WeasyPrint via pip
RUN pip install --no-cache-dir weasyprint

WORKDIR /root/

# Copy compiled Go server binary from build stage
COPY --from=build /app/server .

# Copy templates folder from build stage
COPY --from=build /app/pkg/reports/templates ./pkg/reports/templates

EXPOSE 8080

CMD ["./server"]
