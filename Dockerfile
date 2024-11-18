# Use a base Go image with Debian for better compatibility
FROM golang:1.23-bullseye

# Install necessary dependencies
RUN apt-get update && apt-get install -y \
    chromium \
    chromium-driver \
    libnss3 \
    fonts-liberation \
    libfontconfig1 \
    wget \
    ca-certificates \
    --no-install-recommends && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

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
