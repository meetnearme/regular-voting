#!/bin/bash
set -euo pipefail

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Installing Go..."

    # Check if running on macOS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # Check if Homebrew is installed
        if ! command -v brew &> /dev/null; then
            echo "Homebrew is not installed. Installing Homebrew..."
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

            # Add Homebrew to PATH for the current session
            eval "$(/opt/homebrew/bin/brew shellenv)"
        fi

        echo "Installing Go using Homebrew..."
        brew install go
    else
        # For non-macOS systems, keep the existing installation method
        # Install Go (this example is for Ubuntu/Debian, adjust for other systems)
        sudo apt-get update
        sudo apt-get install -y golang-go
    fi
else
    echo "Go is already installed."
fi

# Check Go version
go_version=$(go version | awk '{print $3}')
echo "Go version: $go_version"

# Minimum required Go version (adjust as needed)
min_version="go1.23"

if [[ "$go_version" < "$min_version" ]]; then
    echo "Go version is too old. Please upgrade to $min_version or newer."
    exit 1
fi

echo "Go installation complete."

# Check if go.mod file exists
if [ ! -f "go.mod" ]; then
    echo "Initializing Go module..."
    go mod init regular-voting  # Replace 'regular-voting' with your actual project name
fi

# Install dependencies
echo "Installing Go dependencies..."
go mod tidy

echo "Go dependencies installed successfully."

templ generate

echo "Templ templates compiled successfully."

go build

echo "Go binary built successfully."

go run .

