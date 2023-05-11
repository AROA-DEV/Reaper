@echo off

REM Set the URL of the configuration file on GitHub
set "configUrl=https://github.com/AROA-DEV/Reaper/blob/Testing/Config/Reaper-config"

REM Set the SSH server details
set "sshServer=username@example.com"
set "remoteFilePath=/path/to/config.txt"

REM Fetch the configuration file contents from GitHub
for /f "delims=" %%i in ('curl -s "%configUrl%"') do (
    REM Check if the line "Active=true" exists in the GitHub configuration file
    echo %%i | findstr /i "Active=true" >nul
    if %errorlevel% neq 0 (
        REM The line "Active=true" is not found or is set to "Active=false" in the GitHub configuration
        echo GitHub configuration is inactive. Skipping script execution.
        exit /b 1
    )
)

REM Execute the remote command to read the configuration file contents from the SSH server
for /f "delims=" %%i in ('ssh %sshServer% "cat %remoteFilePath%"') do (
    REM Check if the line "Active=true" exists in the SSH server configuration file
    echo %%i | findstr /i "Active=true" >nul
    if %errorlevel% neq 0 (
        REM The line "Active=true" is not found or is set to "Active=false" in the SSH server configuration
        echo SSH server configuration is inactive. Skipping script execution.
        exit /b 1
    )
)

REM Both safety features are set to true, execute your script logic here
echo Both safety features are active. Running the script...
REM Add your script execution code below

exit /b 0