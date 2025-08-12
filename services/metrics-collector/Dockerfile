# The official Go image. This image contains all the tools needed to build our app.
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies.
# This layer is cached by Docker, speeding up future builds.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application.
# CGO_ENABLED=0 creates a statically-linked binary.
# -o ./out/agent builds the output to a specific folder.
RUN CGO_ENABLED=0 go build -o ./out/agent .


# --- The Final Image ---
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy ONLY the compiled binary from the 'builder' stage.
COPY --from=builder /app/out/agent .

# Expose port 9100 to the network.
EXPOSE 9100

# The command to run when the container starts.
ENTRYPOINT ["./agent"]