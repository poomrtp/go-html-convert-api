# Use a base image with Go and necessary build tools
FROM golang:1.23-alpine AS builder

# Set working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 is important for static binaries and easier deployment
# -a -installsuffix cgo: Ensures static binary if CGO is enabled
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-api-server

# Use a minimal base image for the final runtime
FROM debian:bookworm-slim

# Install Chromium and other necessary fonts/libraries for chromedp
# These packages are crucial for chromedp to function correctly
RUN apt-get update && apt-get install -y \
    chromium \
    fonts-noto-color-emoji \
    fonts-ipafont-gothic \
    fonts-wqy-zenhei \
    fonts-thai-tlwg \
    libnss3 \
    libatk-bridge2.0-0 \
    libcups2 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libxkbcommon0 \
    libasound2 \
    libgconf-2-4 \
    libu2f-udev \
    libglib2.0-0 \
    libpangocairo-1.0-0 \
    libgtk-3-0 \
    libappindicator3-1 \
    libevent-pthreads-2.1-7 \
    libvulkan1 \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

# Set environment variables for Chromium
ENV CHROME_BIN=/usr/bin/chromium
ENV CHROME_PATH=/usr/bin/chromium
# Ensure headless mode for server environments
ENV CHROMEDP_HEADLESS_MODE=true
# Set timezone if needed
# ENV TZ=Asia/Bangkok 

# Copy the built Go application from the builder stage
COPY --from=builder /go-api-server /go-api-server

# Expose the port (Cloud Run will use $PORT env var)
EXPOSE 8080

# Command to run the application
CMD ["/go-api-server"]