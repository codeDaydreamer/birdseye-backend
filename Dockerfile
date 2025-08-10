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

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the compiled binary
COPY --from=build /app/server .

# Copy the templates folder from build context into the container
COPY --from=build /app/pkg/reports/templates ./pkg/reports/templates

EXPOSE 8080

CMD ["./server"]
