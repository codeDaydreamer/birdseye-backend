# Stage 1: Build
FROM golang:1.23.3-alpine AS build

# Install necessary tools
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first to leverage caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the full project
COPY . .

# Build the Go app (update path to your main file if different)
RUN go build -o server ./cmd/birdseye/main.go

# Stage 2: Run
FROM alpine:latest

# Install necessary system packages (certs + timezone)
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /root/

# Copy the compiled binary from builder
COPY --from=build /app/server .

# Copy wait script
COPY wait-for-mysql.sh /wait-for-mysql.sh
RUN chmod +x /wait-for-mysql.sh

# Expose the port your Go server uses
EXPOSE 8080

# Run the app
CMD ["/wait-for-mysql.sh","./server"]
