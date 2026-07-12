# =============================================================================
# EasyMail control script — Windows (PowerShell)
# Pre-flight checks before launching, supports start / stop / status.
#
# Usage:
#   .\easymail.ps1 start                # start the service
#   .\easymail.ps1 stop                 # stop the service
#   .\easymail.ps1 status               # check running state
#   $env:EASYMAIL_HOME="D:\app"; .\easymail.ps1 start   # custom path
# =============================================================================

param(
    [ValidateSet("start", "stop", "status", "restart")]
    [string]$Command = "start",

    [string]$EASYMAIL_HOME = $PSScriptRoot
)

$CONFIG = Join-Path $EASYMAIL_HOME "config\easymail.yaml"
$BINARY = Join-Path $EASYMAIL_HOME "bin\easymail.exe"

function info  { Write-Host "[INFO]  $args" -ForegroundColor Green }
function warn  { Write-Host "[WARN]  $args" -ForegroundColor Yellow }
function fatal { Write-Host "[FATAL] $args" -ForegroundColor Red; exit 1 }

# ---------------------------------------------------------------------------
# status
# ---------------------------------------------------------------------------
function Get-Status {
    $proc = Get-Process -Name "easymail" -ErrorAction SilentlyContinue
    if ($proc) {
        return $proc
    }
    return $null
}

function Write-Status {
    $proc = Get-Status
    if ($proc) {
        Write-Host "EasyMail is running (PID $($proc.Id))"
        return $true
    } else {
        Write-Host "EasyMail is not running"
        return $false
    }
}

# ---------------------------------------------------------------------------
# start
# ---------------------------------------------------------------------------
function Start-EasyMail {
    $proc = Get-Status
    if ($proc) {
        fatal "EasyMail is already running (PID $($proc.Id)). Stop it first."
    }

    info "EasyMail pre-flight checks ..."

    # 1. Config
    if (-not (Test-Path $CONFIG)) {
        fatal "Config file not found: $CONFIG"
    }
    info "  Config: $CONFIG"

    # 2. Binary
    if (-not (Test-Path $BINARY)) {
        fatal "Binary not found: $BINARY"
    }
    info "  Binary: $BINARY"

    # 3. Directories
    foreach ($dir in @("logs", "storage")) {
        $path = Join-Path $EASYMAIL_HOME $dir
        if (-not (Test-Path $path)) {
            warn "  Directory missing, creating: $path"
            New-Item -ItemType Directory -Path $path -Force | Out-Null
        } else {
            info "  Directory: $path"
        }
    }

    # 4. Optional: ONNX Runtime library (for BERT classify model)
    $onnxLib = Get-ChildItem "$EASYMAIL_HOME\lib\onnxruntime*" -ErrorAction SilentlyContinue
    if ($onnxLib) {
        info "  ONNX Runtime lib: $($onnxLib[0].Name)"
    } else {
        warn "  ONNX Runtime lib not found — BERT classify model unavailable"
    }

    # 5. Optional: FastText executable (for FastText model training)
    $ftBin = Get-ChildItem "$EASYMAIL_HOME\bin\fasttext*" -ErrorAction SilentlyContinue
    if ($ftBin) {
        info "  FastText: $($ftBin[0].Name)"
    } else {
        warn "  FastText executable not found — FastText model training unavailable"
    }

    # 6. Launch
    info "Starting EasyMail ..."
    Set-Location $EASYMAIL_HOME

    try {
        $p = Start-Process -FilePath $BINARY -ArgumentList "-config", $CONFIG -NoNewWindow -PassThru
        Start-Sleep -Seconds 1
        if (-not $p.HasExited) {
            info "EasyMail started (PID $($p.Id))"
        } else {
            fatal "EasyMail failed to start. Exit code: $($p.ExitCode)"
        }
    }
    catch {
        fatal "Failed to start EasyMail: $_"
    }
}

# ---------------------------------------------------------------------------
# stop
# ---------------------------------------------------------------------------
function Stop-EasyMail {
    $proc = Get-Status
    if (-not $proc) {
        warn "EasyMail is not running"
        return
    }

    info "Stopping EasyMail (PID $($proc.Id)) ..."
    taskkill /PID $($proc.Id) 2>$null
    Start-Sleep -Seconds 2

    if (Get-Process -Id $proc.Id -ErrorAction SilentlyContinue) {
        warn "Waiting for graceful shutdown ..."
        $proc.WaitForExit(30000) | Out-Null
    }

    if (-not $proc.HasExited) {
        warn "Graceful shutdown timed out, force killing ..."
        taskkill /F /PID $($proc.Id) 2>$null
    }

    info "EasyMail stopped"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
switch ($Command) {
    "start"   { Start-EasyMail }
    "stop"    { Stop-EasyMail }
    "status"  { Write-Status | Out-Null }
    "restart" { Stop-EasyMail; Start-EasyMail }
}