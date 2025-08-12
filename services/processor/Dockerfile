FROM golang:1.24-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o ./out/processor .


# --- The Final Image ---
FROM alpine:latest

WORKDIR /app

# Copy ONLY the compiled binary from the 'builder' stage
COPY --from=builder /app/out/processor .

# The command to run when the container starts
ENTRYPOINT ["./processor"]