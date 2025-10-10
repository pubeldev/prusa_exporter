@echo off
setlocal enabledelayedexpansion

echo Detecting host IP address...

REM Get all IPv4 addresses and filter out localhost, Docker, and virtual interfaces
for /f "tokens=2 delims=:" %%i in ('ipconfig ^| findstr /R "IPv4.*Address"') do (
    set "ip=%%i"
    REM Remove leading spaces
    set "ip=!ip: =!"
    
    REM Skip localhost
    if not "!ip!"=="127.0.0.1" (
        REM Skip Docker bridge networks (172.17.x.x - 172.31.x.x range)
        echo !ip! | findstr /R "^172\.1[7-9]\." >nul
        if errorlevel 1 (
            echo !ip! | findstr /R "^172\.2[0-9]\." >nul
            if errorlevel 1 (
                echo !ip! | findstr /R "^172\.3[0-1]\." >nul
                if errorlevel 1 (
                    REM Skip other common Docker/VM ranges
                    echo !ip! | findstr /R "^192\.168\.65\." >nul
                    if errorlevel 1 (
                        echo !ip! | findstr /R "^10\.0\.75\." >nul
                        if errorlevel 1 (
                            if not defined HOST_IP (
                                set "HOST_IP=!ip!"
                            )
                        )
                    )
                )
            )
        )
    )
)

if not defined HOST_IP (
    echo Could not find a valid host IP address.
    pause
    exit /b 1
)

echo Using host IP: %HOST_IP%

REM Set the HOST_IP environment variable for docker-compose
set HOST_IP=%HOST_IP%

docker compose up