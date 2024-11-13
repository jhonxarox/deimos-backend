# Use the official Go image with the necessary version
FROM golang:1.23-alpine

# Install necessary dependencies
RUN apk add --no-cache gcc musl-dev chromium

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go app
RUN go build -tags netgo -ldflags '-s -w' -o app

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./app"]
