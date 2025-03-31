@echo off
echo Starting Reaper...

REM Store the start time to create a unique log file
set timestamp=%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%%time:~6,2%
set timestamp=%timestamp: =0%

REM Create a directory for logs if it doesn't exist
if not exist "logs" mkdir logs

REM Run the program and store the process ID
start /b "" go run main.go > logs\reaper_%timestamp%.log 2>&1
for /f "tokens=2" %%a in ('tasklist ^| find "main.exe"') do set PID=%%a

echo Reaper is running with PID %PID%
echo Press Ctrl+C to stop and cleanup...

:WAIT
timeout /t 1 /nobreak > nul
tasklist | find "%PID%" > nul
if errorlevel 1 goto CLEANUP
goto WAIT

:CLEANUP
echo Cleaning up...

REM Kill any remaining program processes
taskkill /F /IM main.exe 2>nul
taskkill /F /FI "WINDOWTITLE eq indexing.log" 2>nul

REM Remove program files
if exist "files.db" del /F /Q "files.db"
if exist "indexing.log" del /F /Q "indexing.log"

REM Move the current run log to logs directory
move /Y "*.log" "logs\" 2>nul

echo Cleanup complete!
exit
