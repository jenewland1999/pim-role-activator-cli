# Install on Windows

## Prerequisites

- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli-windows) — required for authentication
- [Go 1.22+](https://go.dev/dl/) — required to build from source

### Install Prerequisites with winget

```powershell
winget install Microsoft.AzureCLI
winget install GoLang.Go
```

## Option A: Download a Pre-Built Binary

Download the latest release for Windows from the [Releases](https://github.com/jenewland1999/pim-role-activator-cli/releases/latest) page.

**PowerShell:**

```powershell
# Download
Invoke-WebRequest -Uri "https://github.com/jenewland1999/pim-role-activator-cli/releases/latest/download/pim-windows-amd64.exe" -OutFile "pim.exe"

# Move to a directory in your PATH (create it if needed)
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\bin"
Move-Item -Force pim.exe "$env:USERPROFILE\bin\pim.exe"
```

Add `%USERPROFILE%\bin` to your system `PATH` if it isn't already:

```powershell
# Add to PATH for current user (persistent)
[Environment]::SetEnvironmentVariable("Path", "$env:Path;$env:USERPROFILE\bin", "User")
```

Restart your terminal for the PATH change to take effect.

## Option B: Install with `go install`

```powershell
go install github.com/jenewland1999/pim-role-activator-cli/cmd/pim@latest
```

This places the binary in `%GOPATH%\bin` (usually `%USERPROFILE%\go\bin`). Make sure that directory is in your `PATH`.

## Option C: Build from Source

```powershell
git clone https://github.com/jenewland1999/pim-role-activator-cli.git
cd pim-role-activator-cli
go build -o pim.exe ./cmd/pim
Move-Item -Force pim.exe "$env:USERPROFILE\bin\pim.exe"
```

## Verify Installation

```powershell
pim --help
```

## First-Time Setup

1. Log in to Azure:

   ```powershell
   az login
   ```

2. Run `pim` — the setup wizard will launch automatically on first run and configure your subscriptions and identity.

## Uninstall

```powershell
Remove-Item "$env:USERPROFILE\bin\pim.exe"
Remove-Item -Recurse "$env:USERPROFILE\.pim"
```
