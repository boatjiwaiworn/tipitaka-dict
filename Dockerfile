# Stage 1: Build the frontend (Vite)
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

# Stage 2: Build the Go backend
FROM golang:1.22-alpine AS backend-builder
# Install build dependencies for CGO (sqlite3)
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
# First copy go.mod and go.sum to cache dependencies
COPY server/go.mod server/go.sum ./server/
WORKDIR /app/server
RUN go mod download
# Copy the rest of the server source code
WORKDIR /app
COPY server ./server
# Build the application
WORKDIR /app/server
RUN CGO_ENABLED=1 GOOS=linux go build -o tipitaka_app main.go

# Stage 3: Final lightweight container
FROM alpine:latest
WORKDIR /app
# Copy the compiled binary from the backend builder stage
COPY --from=backend-builder /app/server/tipitaka_app .
# Copy the built frontend from the frontend builder stage
COPY --from=frontend-builder /app/dist ./dist
# Copy the required data and static files
COPY server-data ./server-data

# Expose the port that the application listens on
EXPOSE 8402

# Run the application (no-open flag prevents it from trying to open a browser)
CMD ["./tipitaka_app", "-no-open"]
