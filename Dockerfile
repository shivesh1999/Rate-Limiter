# Use the official Golang image as the base image
FROM golang:1.24 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the entire project
COPY . .

# Build the Go application
RUN go build -o rate-limiter .

# Use a smaller base image for running the application
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/rate-limiter .

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./rate-limiter"]
