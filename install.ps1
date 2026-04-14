# Siby-Terminal Installer for Windows
# Run: powershell -ExecutionPolicy Bypass -File install.ps1

$ErrorActionPreference = "Stop"

function Write-Banner {
    Write-Host ""
    Write-Host "╔══════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║    Siby-Terminal Installer v0.1.0        ║" -ForegroundColor Cyan
    Write-Host "╚══════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
}

function Test-GoInstalled {
    try {
        $goVersion = go version 2>$null
        if ($goVersion) {
            Write-Host "[OK] Go installed: $goVersion" -ForegroundColor Green
            return $true
        }
    }
    catch { }
    Write-Host "[FAIL] Go not found" -ForegroundColor Red
    Write-Host "  Download: https://go.dev/dl/" -ForegroundColor Yellow
    return $false
}

function Test-OllamaRunning {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:11434/api/tags" -UseBasicParsing -TimeoutSec 2 -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            Write-Host "[OK] Ollama running on localhost:11434" -ForegroundColor Green
            return $true
        }
    }
    catch { }
    Write-Host "[--] Ollama not running (optional)" -ForegroundColor Yellow
    Write-Host "  Download: https://ollama.ai/download" -ForegroundColor Gray
    return $false
}

function Test-ApiKeys {
    if ($env:ANTHROPIC_API_KEY) {
        Write-Host "[OK] ANTHROPIC_API_KEY detected" -ForegroundColor Green
    }
    if ($env:OPENAI_API_KEY) {
        Write-Host "[OK] OPENAI_API_KEY detected" -ForegroundColor Green
    }
}

function Install-Siby {
    Write-Host ""
    Write-Host "Installing Siby-Terminal..." -ForegroundColor Cyan
    
    $InstallDir = "$env:USERPROFILE\.local\bin"
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    
    $Version = "0.1.0"
    $DownloadUrl = "https://github.com/siby-agentiq/siby-terminal/releases/download/v${Version}/siby-windows-amd64.exe"
    $OutFile = "$InstallDir\siby.exe"
    
    try {
        Write-Host "  Downloading..." -ForegroundColor Gray
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $OutFile -UseBasicParsing
        Write-Host "  [OK] Downloaded to $OutFile" -ForegroundColor Green
    }
    catch {
        Write-Host "  [OK] Building from source..." -ForegroundColor Yellow
        Set-Location $PSScriptRoot
        go build -ldflags="-s -w" -o $OutFile .\cmd\siby-agentiq
    }
    
    $PathEnv = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($PathEnv -notlike "*$InstallDir*") {
        Write-Host ""
        Write-Host "Add to PATH:" -ForegroundColor Yellow
        Write-Host "  \$env:Path += ';$InstallDir'" -ForegroundColor Gray
    }
    
    Write-Host "  [OK] Installed to $OutFile" -ForegroundColor Green
    return $OutFile
}

function Setup-Config {
    Write-Host ""
    Write-Host "Creating config..." -ForegroundColor Cyan
    
    $ConfigDir = "$env:USERPROFILE\.config\siby"
    if (-not (Test-Path $ConfigDir)) {
        New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
    }
    
    $ConfigFile = "$ConfigDir\config.yaml"
    if (-not (Test-Path $ConfigFile)) {
        @"
version: "1.0"
providers:
  primary: "ollama"
  ollama:
    enabled: true
    base_url: "http://localhost:11434"
    default_model: "llama3.2:latest"
  anthropic:
    enabled: true
    api_key: "`${ANTHROPIC_API_KEY}"
  openai:
    enabled: true
    api_key: "`${OPENAI_API_KEY}"
chain_of_thought:
  enabled: true
  reasoning_depth: "deep"
"@ | Out-File -FilePath $ConfigFile -Encoding UTF8
        Write-Host "  [OK] Config created at $ConfigFile" -ForegroundColor Green
    }
}

Write-Banner

if (-not (Test-GoInstalled)) {
    exit 1
}

Test-OllamaRunning
Test-ApiKeys
$SibyPath = Install-Siby
Setup-Config

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Run: $SibyPath" -ForegroundColor Cyan
Write-Host "Or: siby" -ForegroundColor Cyan
Write-Host ""
