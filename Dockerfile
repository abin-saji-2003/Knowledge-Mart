# Use official Golang image as base
FROM golang:1.22 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project files
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Use a minimal Alpine image for the final stage
FROM alpine:latest

# Set the working directory to /app inside the container
WORKDIR /app

# Install necessary dependencies (Alpine needs this for networking and execution)
RUN apk add --no-cache ca-certificates libc6-compat

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy the .env file (if you have one) to the correct directory
COPY .env /app/.env

# Copy the templates directory
COPY templates/ /app/templates/

# Expose the port your app runs on
EXPOSE 8080

# Command to run the application
CMD ["./main"]
