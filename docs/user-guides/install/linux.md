# Install on Linux

## Prerequisites

- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli-linux) — required for authentication
- [Go 1.22+](https://go.dev/dl/) — required to build from source

### Install Prerequisites

**Debian / Ubuntu:**

```bash
# Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Go (check https://go.dev/dl/ for the latest version)
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
export PATH="$PATH:/usr/local/go/bin"
```

**Fedora / RHEL:**

```bash
# Azure CLI
sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc
sudo dnf install azure-cli

# Go
sudo dnf install golang
```

## Option A: Download a Pre-Built Binary

Download the latest release for Linux from the [Releases](https://github.com/jenewland1999/pim-role-activator-cli/releases/latest) page.

**x86_64 (amd64):**

```bash
curl -Lo pim https://github.com/jenewland1999/pim-role-activator-cli/releases/latest/download/pim-linux-amd64
chmod +x pim
sudo mv pim /usr/local/bin/pim
```

**ARM64 (aarch64):**

```bash
curl -Lo pim https://github.com/jenewland1999/pim-role-activator-cli/releases/latest/download/pim-linux-arm64
chmod +x pim
sudo mv pim /usr/local/bin/pim
```

## Option B: Install with `go install`

```bash
go install github.com/jenewland1999/pim-role-activator-cli/cmd/pim@latest
```

This places the binary in `$GOPATH/bin` (usually `~/go/bin`). Make sure that directory is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Add the line above to your `~/.bashrc` or `~/.zshrc` to make it permanent.

## Option C: Build from Source

```bash
git clone https://github.com/jenewland1999/pim-role-activator-cli.git
cd pim-role-activator-cli
go build -o pim ./cmd/pim
sudo mv pim /usr/local/bin/pim
```

## Verify Installation

```bash
pim --help
```

## First-Time Setup

1. Log in to Azure:

   ```bash
   az login
   ```

   On a headless server (no browser):

   ```bash
   az login --use-device-code
   ```

2. Run `pim` — the setup wizard will launch automatically on first run and configure your subscriptions and identity.

## Uninstall

```bash
sudo rm /usr/local/bin/pim
rm -rf ~/.pim
```
