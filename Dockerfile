# Stage 1: Build
FROM golang:1.23.3-alpine AS build

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server ./cmd/birdseye/main.go

# Stage 2: Run
FROM alpine:latest

# Install dependencies for WeasyPrint + ca-certificates + tzdata + python3 + pip
RUN apk add --no-cache \
    python3 \
    py3-pip \
    cairo-dev \
    pango-dev \
    gdk-pixbuf-dev \
    libffi-dev \
    musl-dev \
    build-base \
    cairo \
    pango \
    gdk-pixbuf \
    ca-certificates \
    tzdata

# Install WeasyPrint via pip
RUN pip3 install --no-cache-dir weasyprint

WORKDIR /root/

# Copy the compiled binary
COPY --from=build /app/server .

# Copy the templates folder from build stage
COPY --from=build /app/pkg/reports/templates ./pkg/reports/templates

EXPOSE 8080

CMD ["./server"]
