# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# Stage 2: Create a minimal image with the Go binary
FROM scratch

# Copy the Go binary from the builder stage
COPY --from=builder /app/main /main

# Set the entrypoint to the Go binary
ENTRYPOINT ["/main"]