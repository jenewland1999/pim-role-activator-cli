# Install on macOS

## Prerequisites

- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli-macos) — required for authentication
- [Go 1.25+](https://go.dev/dl/) — required to build from source

### Install Prerequisites with Homebrew

```bash
brew install azure-cli go
```

## Option A: Download a Pre-Built Binary

Download the latest release for macOS from the [Releases](https://github.com/jenewland1999/pim-role-activator-cli/releases/latest) page.

**Apple Silicon (M1/M2/M3/M4):**

```bash
# Download
curl -Lo pim https://github.com/jenewland1999/pim-role-activator-cli/releases/latest/download/pim-darwin-arm64

# Make executable
chmod +x pim

# Move to PATH
sudo mv pim /usr/local/bin/pim
```

**Intel:**

```bash
curl -Lo pim https://github.com/jenewland1999/pim-role-activator-cli/releases/latest/download/pim-darwin-amd64
chmod +x pim
sudo mv pim /usr/local/bin/pim
```

> **macOS Gatekeeper:** If macOS blocks the binary ("cannot be opened because the developer cannot be verified"), run:
>
> ```bash
> xattr -d com.apple.quarantine /usr/local/bin/pim
> ```

## Option B: Install with `go install`

```bash
go install github.com/jenewland1999/pim-role-activator-cli/cmd/pim@latest
```

This places the binary in `$GOPATH/bin` (usually `~/go/bin`). Make sure that directory is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Add the line above to your `~/.zshrc` or `~/.bash_profile` to make it permanent.

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

## Setup

1. Log in to Azure:

   ```bash
   az login
   ```

2. Run `pim` — the setup wizard will launch automatically on first run and configure your subscriptions and identity.

## Usage

```bash
# Show active roles
pim

# Activate roles interactively
pim activate

# Dry run activation flow
pim activate --dry-run

# Re-run setup manually
pim setup
```

## Uninstall

```bash
sudo rm /usr/local/bin/pim
rm -rf ~/.pim
```
