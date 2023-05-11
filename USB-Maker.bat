@echo off
echo.
echo "USB Reaper creator"
echo.
set /p usb="What is the external drive letter? "

:one
xcopy  /v /s /e /h /i /y "Reaper-Ultimate.bat" "%usb%:\"