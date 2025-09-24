# Use distroless as minimal base image to package the application
# Distroless images contain only your application and its runtime dependencies
FROM golang:1.24.6-bullseye AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the entire project
COPY . .

# Build the Go application
# Adding security flags
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o rate-limiter .

# Use Google's distroless base image to reduce attack surface
# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/base:nonroot

# Create directory for the app
WORKDIR /app

# Copy the compiled binary and .env file from the builder stage
COPY --from=builder /app/rate-limiter .
COPY --from=builder /app/.env .

# Run as non-root user
USER nonroot:nonroot

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./rate-limiter"]