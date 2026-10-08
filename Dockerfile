# Stage 1: build the server binary using the Go toolchain on Alpine.
FROM golang:1.22-alpine AS builder

# Work inside /src for all following instructions.
WORKDIR /src

# Copy only the module files first so the dependency layer is cached
# until go.mod or go.sum change.
COPY go.mod go.sum ./

# Download dependencies into the module cache.
RUN go mod download

# Copy the rest of the source code.
COPY . .

# Compile the server into /out/server as a static binary. The output is not
# written to ./server because that is the source directory, and go build
# would place the binary inside it.
RUN CGO_ENABLED=0 go build -o /out/server ./server

# Stage 2: minimal runtime image.
FROM alpine:latest

# The server serves ./static relative to its working directory.
WORKDIR /app

# Copy only the compiled binary from the builder stage.
COPY --from=builder /out/server ./server

# Copy the browser client files from the builder stage.
COPY --from=builder /src/static ./static

# Document the default port. The server reads the PORT environment
# variable at runtime, which Railway sets automatically (default 8080).
EXPOSE 8080

# Start the server when the container runs.
ENTRYPOINT ["./server"]
