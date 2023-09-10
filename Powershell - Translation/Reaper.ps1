# Set the URL of the configuration file on GitHub
# There is an empty template at https://raw.githubusercontent.com/raf181/Config/main/Reaper/empty-config.cfg
$configUrl = "https://raw.githubusercontent.com/raf181/Config/main/Reaper/config.cfg"
$configLocal = "Local-Config.cfg"

# Initialize availableRemoteConfig and activeStatus to expected values
$availableRemoteConfig = $null
$activeStatus = $null

# Fetch and process the configuration file
$configContent = Invoke-RestMethod -Uri $configUrl -ErrorAction SilentlyContinue
if ($?) {
    $availableRemoteConfig = $true
    $activeStatus = ($configContent -match "Active=true")
}
else {
    $availableRemoteConfig = $false
}

# Process based on fetched configuration
if ($availableRemoteConfig) {
    if ($activeStatus) {
        $configFile = Invoke-RestMethod -Uri $configUrl
    }
    else {
        Write-Host "Remote file available with 'Active=false'."
        Read-Host -Prompt "Press Enter to continue..."
        exit
    }
}
else {
    if (Test-Path $configLocal) {
        $localConfigContent = Get-Content -Path $configLocal
        if ($localConfigContent -match "Active=true") {
            $configFile = Get-Content -Path $configLocal
        }
        else {
            Write-Host "Local configuration file is not active. Exiting..."
        }
    }
    else {
        Write-Host "Local configuration file not found. Exiting..."
    }
    Read-Host -Prompt "Press Enter to continue..."
    exit
}

Read-Host "Press Enter to continue..."
# Initialize variables
$robo_flags = $null
$ssh_copy = $null
$ssh_user = $null
$target_server = $null
$target_port = $null
$target_folder = $null

# Get variables from config file
foreach ($line in $configFile) {
    $key, $value = $line -split '=', 2
    switch ($key.Trim().ToLower()) {
        "robo_flags" {
            $roboflags = $value.Trim()
        }
        "ssh_copy" {
            $sshcopy = $value.Trim()
        }
        "ssh_user" {
            $sshuser = $value.Trim()
        }
        "target_server" {
            $targetserver = $value.Trim()
        }
        "target_port" {
            $targetport = $value.Trim()
        }
        "target_folder" {
            $targetfolder = $value.Trim()
        }
    }
}
Write-Host "robo_flags: $roboflags"
Write-Host "ssh_copy: $sshcopy"
Write-Host "ssh_user: $sshuser"
Write-Host "target_server: $targetserver"
Write-Host "target_port: $targetport"
Write-Host "target_folder: $targetfolder"

# Check if text to speech is enabled
if ($null -eq $roboflags) {
    $roboflags = "/E /COPY:DAT /NP /NFL /NDL /NJH /NJS /NC /NS /MT:32 /TEE /R:5 /W:10 /BYTES"
    Write-Host "robo_flags not found in config file. Using default flags: $roboflags"
}
if ($null -eq $sshcopy) {
    $sshcopy = "false"
    Write-Host "ssh_copy not found in config file. Using default value: $sshcopy"
}
if ($null -eq $sshuser) {
    $sshuser = "user"
    Write-Host "ssh_user not found in config file. Using default value: $sshuser"
}
if ($null -eq $targetserver) {
    $targetserver = "server"
    Write-Host "target_server not found in config file. Using default value: $targetserver"
}
if ($null -eq $targetport) {
    $targetport = "22"
    Write-Host "target_port not found in config file. Using default value: $targetport"
}
if ($null -eq $targetfolder) {
    $targetfolder = "/home/user"
    Write-Host "target_folder not found in config file. Using default value: $targetfolder"
}



# Automatically set the USB drive letter
$usb_drive = (Get-Location).Path
$ssh_key = "$env:USERPROFILE\.ssh\"
$desktop = "$env:USERPROFILE\Desktop\"
$documents = "$env:USERPROFILE\Documents\"
$images = "$env:USERPROFILE\Pictures\"
$downloads = "$env:USERPROFILE\Downloads\"
$onedrive = "$env:USERPROFILE\OneDrive\"

$dest_folder = "${usb_drive}$env:USERNAME"

# Clear-Host
Write-Host ""
Write-Host "Copying files to USB ($usb_drive)..."
Write-Host "Destination folder: $dest_folder"
Write-Host ""
Read-Host "Press Enter to continue..."

# Remove -PassThru to see the files being copied

Write-Host "Start copying .ssh"
Robocopy $ssh_key "$dest_folder\.ssh\" $roboflags
Write-Host ".ssh copied"

#Write-Host "Start copying OneDrive"
#Robocopy $onedrive "$dest_folder\OneDrive\" $robo_flags
#Write-Host "OneDrive copied"

Write-Host "Start copying Desktop"
Robocopy $desktop "$dest_folder\Desktop\" $roboflags
Write-Host "Desktop copied"

Write-Host "Start copying Documents"
Robocopy $documents "$dest_folder\Documents\" $roboflags
Write-Host "Documents copied"

Write-Host "Start copying Images"
Robocopy $images "$dest_folder\Images\" $roboflags
Write-Host "Images copied"

Write-Host "Start copying Downloads"
Robocopy $downloads "$dest_folder\Downloads\" $roboflags
Write-Host "Downloads copied"

Write-Host "Files copied successfully to USB ($usb_drive)!"
Pause
Clear-Host

exit