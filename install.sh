#!/bin/bash
set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}"
cat << 'EOF'
   _____ _____ ____  __  __   ___ _    _____ _  __
  |_   _| ____|  _ \|  \/  | |_ _|__|___ /| |/ /___ _   _____ _ __ 
    | | |  _| | |_) | |\/| |  | |/ __|_ \| ' // _ \ | / / _ \ '__|
    | | | |___|  _ <| |  | |  | |\__ \__) | . \  __/ | \ \  __/ |   
    |_| |_____|_| \_\_|  |_| |___|___/____|_|\_\___|_| \_/\___|_|   
                                                                      
                     The Last Agent You Will Ever Need
EOF
echo -e "${NC}"

INSTALL_DIR="${HOME}/.local/bin"
BINARY="${INSTALL_DIR}/siby"

echo -e "${BOLD}[1/4]${NC} Checking system..."
if ! command -v go &> /dev/null && ! command -v curl &> /dev/null; then
    echo -e "${RED}✗${NC} Go or curl required"
    exit 1
fi
echo -e "${GREEN}✓${NC} System ready"

echo -e "${BOLD}[2/4]${NC} Installing Siby..."
mkdir -p "$INSTALL_DIR"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

RELEASE_URL="https://github.com/siby-agentiq/siby-terminal/releases/latest/download"
TARBALL="siby-${OS}-${ARCH}.tar.gz"

if command -v curl &> /dev/null; then
    curl -fsSL "${RELEASE_URL}/${TARBALL}" -o "/tmp/siby.tar.gz" 2>/dev/null || build_from_source
else
    wget -q "${RELEASE_URL}/${TARBALL}" -O "/tmp/siby.tar.gz" 2>/dev/null || build_from_source
fi

if [ -f "/tmp/siby.tar.gz" ]; then
    tar -xzf "/tmp/siby.tar.gz" -C "$INSTALL_DIR"
    chmod +x "$BINARY"
    rm -f "/tmp/siby.tar.gz"
    echo -e "${GREEN}✓${NC} Binary installed to ${BINARY}"
else
    build_from_source
fi

echo -e "${BOLD}[3/4]${NC} Configuring shell..."

SHELL_RC="${HOME}/.bashrc"
if [ -n "$ZSH_VERSION" ]; then
    SHELL_RC="${HOME}/.zshrc"
fi

if ! grep -q "alias siby=" "$SHELL_RC" 2>/dev/null; then
    echo 'alias siby="siby"' >> "$SHELL_RC"
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$SHELL_RC"
    echo -e "${GREEN}✓${NC} Alias 'siby' added to ${SHELL_RC}"
fi

echo -e "${BOLD}[4/4]${NC} Testing providers..."

test_provider() {
    local name=$1
    local url=$2
    if curl -s --max-time 2 "$url" &>/dev/null; then
        echo -e "  ${GREEN}✓${NC} ${name} running"
        return 0
    else
        echo -e "  ${YELLOW}○${NC} ${name} not found"
        return 1
    fi
}

test_provider "Ollama" "http://localhost:11434/api/tags"

if [ -n "$GROQ_API_KEY" ]; then
    echo -e "  ${GREEN}✓${NC} Groq API key detected"
fi

if [ -n "$ANTHROPIC_API_KEY" ]; then
    echo -e "  ${GREEN}✓${NC} Anthropic API key detected"
fi

echo ""
echo -e "${GREEN}${BOLD}Siby-Agentiq installed successfully!${NC}"
echo ""
echo -e "Run: ${CYAN}siby${NC}"
echo -e "Or:  ${CYAN}${BINARY}${NC}"
echo ""
echo -e "${YELLOW}First time? Install a model:${NC}"
echo -e "  ${BOLD}ollama pull llama3.2${NC}"
echo ""

build_from_source() {
    echo -e "${YELLOW}Building from source...${NC}"
    if ! command -v go &> /dev/null; then
        echo -e "${RED}✗ Go required for build from source${NC}"
        echo -e "Install: ${CYAN}https://go.dev/dl/${NC}"
        exit 1
    fi
    
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [ -f "${SCRIPT_DIR}/go.mod" ]; then
        cd "$SCRIPT_DIR"
        go build -ldflags="-s -w" -o "$BINARY" ./cmd/siby-agentiq
    else
        echo -e "${RED}✗ Source not found${NC}"
        exit 1
    fi
}
