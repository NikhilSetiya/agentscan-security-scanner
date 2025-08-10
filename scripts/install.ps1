# AgentScan CLI Installation Script for Windows
# Usage: Invoke-WebRequest -Uri https://install.agentscan.dev/windows -OutFile install.ps1; .\install.ps1

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:USERPROFILE\.agentscan\bin"
)

$ErrorActionPreference = "Stop"

# Colors for output
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Blue"
}

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Log-Info {
    param([string]$Message)
    Write-ColorOutput "[INFO] $Message" $Colors.Blue
}

function Log-Success {
    param([string]$Message)
    Write-ColorOutput "[SUCCESS] $Message" $Colors.Green
}

function Log-Warning {
    param([string]$Message)
    Write-ColorOutput "[WARNING] $Message" $Colors.Yellow
}

function Log-Error {
    param([string]$Message)
    Write-ColorOutput "[ERROR] $Message" $Colors.Red
}

function Get-Platform {
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
    return "windows_$arch"
}

function Get-LatestVersion {
    if ($Version -eq "latest") {
        Log-Info "Fetching latest version..."
        try {
            $response = Invoke-RestMethod -Uri "https://api.github.com/repos/agentscan/agentscan/releases/latest"
            return $response.tag_name
        }
        catch {
            Log-Warning "Could not fetch latest version, using v1.0.0"
            return "v1.0.0"
        }
    }
    else {
        return $Version
    }
}

function Install-Binary {
    param(
        [string]$Platform,
        [string]$Version
    )
    
    $downloadUrl = "https://github.com/agentscan/agentscan/releases/download/$Version/agentscan-cli-$Platform.zip"
    $tempDir = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString()
    $zipFile = "$tempDir\agentscan-cli.zip"
    
    Log-Info "Installing AgentScan CLI $Version for $Platform..."
    
    try {
        # Create directories
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        
        # Download binary
        Log-Info "Downloading from $downloadUrl..."
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipFile
        
        # Extract binary
        Log-Info "Extracting binary..."
        Expand-Archive -Path $zipFile -DestinationPath $tempDir -Force
        
        # Move binary to install directory
        $binaryPath = "$tempDir\agentscan-cli.exe"
        $installPath = "$InstallDir\agentscan-cli.exe"
        
        if (Test-Path $binaryPath) {
            Move-Item -Path $binaryPath -Destination $installPath -Force
            Log-Success "AgentScan CLI installed to $installPath"
        }
        else {
            throw "Binary not found in extracted files"
        }
    }
    finally {
        # Cleanup
        if (Test-Path $tempDir) {
            Remove-Item -Path $tempDir -Recurse -Force
        }
    }
}

function Setup-Path {
    # Check if already in PATH
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -like "*$InstallDir*") {
        Log-Info "AgentScan CLI is already in PATH"
        return
    }
    
    # Add to user PATH
    $newPath = "$InstallDir;$currentPath"
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    
    # Update current session PATH
    $env:PATH = "$InstallDir;$env:PATH"
    
    Log-Success "Added $InstallDir to PATH"
    Log-Info "PATH updated for current session and future sessions"
}

function Test-Installation {
    $binaryPath = "$InstallDir\agentscan-cli.exe"
    
    if (Test-Path $binaryPath) {
        Log-Success "Installation verified!"
        
        # Test the binary
        try {
            & $binaryPath version | Out-Null
            Log-Info "AgentScan CLI is working correctly"
            & $binaryPath version
        }
        catch {
            Log-Warning "Binary installed but may not be working correctly"
        }
    }
    else {
        Log-Error "Installation failed - binary not found"
        exit 1
    }
}

function Show-Usage {
    Write-Host ""
    Write-ColorOutput "🔒 AgentScan CLI Installation Complete!" $Colors.Green
    Write-Host ""
    Write-Host "Usage:"
    Write-Host "  agentscan-cli scan                    # Run security scan"
    Write-Host "  agentscan-cli scan --help             # Show scan options"
    Write-Host "  agentscan-cli version                 # Show version"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  # Basic scan"
    Write-Host "  agentscan-cli scan"
    Write-Host ""
    Write-Host "  # Scan with API integration"
    Write-Host "  agentscan-cli scan --api-url=https://api.agentscan.dev --api-token=`$env:TOKEN"
    Write-Host ""
    Write-Host "  # Fail on medium or high severity findings"
    Write-Host "  agentscan-cli scan --fail-on-severity=medium"
    Write-Host ""
    Write-Host "  # Exclude specific paths"
    Write-Host "  agentscan-cli scan --exclude-path=node_modules --exclude-path=vendor"
    Write-Host ""
    Write-Host "Documentation: https://docs.agentscan.dev"
    Write-Host "Support: https://github.com/agentscan/agentscan/issues"
}

# Main installation flow
function Main {
    Log-Info "Starting AgentScan CLI installation..."
    
    # Detect platform
    $platform = Get-Platform
    Log-Info "Detected platform: $platform"
    
    # Get version
    $versionToInstall = Get-LatestVersion
    Log-Info "Installing version: $versionToInstall"
    
    # Install binary
    Install-Binary -Platform $platform -Version $versionToInstall
    
    # Setup PATH
    Setup-Path
    
    # Verify installation
    Test-Installation
    
    # Show usage
    Show-Usage
}

# Run main function
try {
    Main
}
catch {
    Log-Error "Installation failed: $($_.Exception.Message)"
    exit 1
}