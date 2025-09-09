# Development container Dockerfile
FROM golang:1.22.2-bookworm

# Install dependencies and development tools
RUN apt-get update && apt-get install -y \
    libusb-1.0-0-dev \
    pkg-config \
    gcc \
    git \
    vim \
    usbutils \
    sudo \
    zsh \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools
RUN go install -v golang.org/x/tools/gopls@latest \
    && go install -v github.com/go-delve/delve/cmd/dlv@latest \
    && go install -v honnef.co/go/tools/cmd/staticcheck@latest

# Create vscode user with sudo privileges
RUN useradd -m -s /bin/zsh -u 1000 vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set up workspace
WORKDIR /workspace

# Keep container running
CMD ["sleep", "infinity"]