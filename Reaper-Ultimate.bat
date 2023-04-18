@echo off
setlocal enabledelayedexpansion

echo .
echo Reaper V-0.1
echo .

set /p usb_drive="Please enter the drive letter of your USB: "
set /p machine_name="Please enter the name of this machine: "

set desktop="%USERPROFILE%\Desktop"
set documents="%USERPROFILE%\Documents"
set images="%USERPROFILE%\Pictures"
set downloads="%USERPROFILE%\Downloads"
set onedrive="%USERPROFILE%\OneDrive"

set dest_folder="%usb_drive%\%machine_name%"

md "%dest_folder%"

xcopy /E /Y "%desktop%" "%dest_folder%\Desktop\"
xcopy /E /Y "%documents%" "%dest_folder%\Documents\"
xcopy /E /Y "%images%" "%dest_folder%\Images\"
xcopy /E /Y "%downloads%" "%dest_folder%\Downloads\"
xcopy /E /Y "%onedrive%" "%dest_folder%\OneDrive\"

echo Files copied successfully!
pause