@echo off
echo.
echo "USB Reaper creator"
echo.
set /p usb="What is the external drive letter? "
xcopy /v /y "Reaper-Ultimate.bat" "%usb%:\"
if %errorlevel%==0 (
  echo "Reaper-Ultimate.bat copied successfully"
) else (
  echo "Error copying Reaper-Ultimate.bat"
)
xcopy /v /y "Config\Local-Config.cfg" "%usb%:\"
if %errorlevel%==0 (
  echo "Local-Config.cfg copied successfully"
) else (
  echo "Error copying Local-Config.cfg"
)