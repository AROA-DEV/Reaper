@echo off
setlocal enabledelayedexpansion

:: Set the URL of the configuration file on GitHub
set "configUrl=https://raw.githubusercontent.com/AROA-DEV/Reaper/Testing/Config/Reaper-config.cfg"

:: Read the remote configuration file
for /f "usebackq tokens=1* delims== " %%a in (`curl -L "%configUrl%"`) do (
    if /i "%%a"=="TARGET_SERVER" (
        :: Get the target server from the remote configuration
        set "target_server=%%b"
    ) else if /i "%%a"=="TARGET_PORT" (
        :: Get the target server port from the remote configuration
        set "target_port=%%b"
    ) else if /i "%%a"=="SSH-USER" (
        :: Get the choice to copy files from the remote configuration
        set "ssh-user=%%b"
    ) else if /i "%%a"=="SSH-PASS" (
        :: Get the choice to copy files from the remote configuration
        set "ssh-pass=%%b"
    ) else if /i "%%a"=="TARGET_FOLDER" (
        :: Get the choice to copy files from the remote configuration
        set "target_folder=%%b"
    )
)

:: Validate if all the required variables are set
if not defined target_server (
    echo TARGET_SERVER is not set in the remote configuration file.
    exit /b 1
)
if not defined target_port (
    echo TARGET_PORT is not set in the remote configuration file. Using default port 22.
    set "target_port=22"  :: Set default port to 22 if not specified in the remote configuration
    echo target port is set to %target_port% by defoult
    pause
)
if not defined ssh-user (
    echo SSH-USER is not set in the remote configuration file.
    set "ssh-user=reaper"  :: Set default user to reaper if not specified in the remote configuration
    echo ssh user is set to %ssh-user% by defoult
    pause
)
if not defined ssh-pass (
    echo SSH-PASS is not set in the remote configuration file.
    pause
    exit /b 1
)
if not defined target_folder (
    echo target_folder is not set in the remote configuration file.
    set "target_folder=~/Reaper-info-retrieve/%USERNAME%/"  :: Set default folder to ~/Reaper-info-retrieve/%USERNAME%/ if not specified in the remote configuration
    echo target folder is set to %target_folder% by defoult
    pause
)

:: Create the user on the SSH server
ssh %ssh-user%@%target_server% -p %target_port% "sudo adduser --disabled-password --gecos '' %username%"
ssh %ssh-user%@%target_server% -p %target_port% "echo %username%:%password% | sudo chpasswd"
:: echo now generate a ssh key
:: pause
:: ssh-keygen
echo User %username% has been created on the SSH server.
