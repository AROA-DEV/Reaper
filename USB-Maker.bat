@echo off
echo.
echo "USB Reaper creator"
echo.
set /p usb="What is the external drive letter? "

:one
xcopy  /E /Y "Reaper-Ultimate.bat" "%usb%:\"