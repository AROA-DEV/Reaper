@echo off
echo.
echo Creating Test enviroment for reaper
echo.
mkdir Test-enviroment
xcopy /y "Reaper-Ultimate.bat" "Test-enviroment"
call Test-enviroment\Reaper-Ultimate.bat
echo.
echo  Test finished
echo.
set /p CopyFiles="Do you want to keep copied files? (y/n): "
if /i "%CopyFiles%"=="y" (
    echo Will remuve the files press ctrl + C
    pause
    cd Test-enviromen
    rmdir /s /q %USERNAME%
    exit
) 
if /i "%CopyFiles%"=="n" (
    exit
)