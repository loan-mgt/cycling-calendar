# Build stage
FROM golang:1.23-alpine AS builder

ENV CGO_ENABLED=0

# Create a working directory
WORKDIR /app

# Copy the Go modules and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary
RUN go build -o /app/cycling-app main.go

# Final stage
FROM golang:1.23-alpine

# Install openssl
RUN apk add --no-cache openssl

# Create a working directory
WORKDIR /root/

# Copy the compiled Go binary from the build stage
COPY --from=builder /app/cycling-app .

# Copy static files to the container
COPY static ./static

# Use a shell form to run both commands sequentially
CMD "/cycling-app"
