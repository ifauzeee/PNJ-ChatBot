@echo off
REM ============================================================
REM 🎭 PNJ Anonymous Bot — Deploy Script (Windows)
REM ============================================================
REM Usage:
REM   deploy.bat          → Build & start (development)
REM   deploy.bat prod     → Build & start (production)
REM   deploy.bat stop     → Stop all containers
REM   deploy.bat logs     → View live logs
REM   deploy.bat status   → Check container status
REM   deploy.bat restart  → Restart containers
REM   deploy.bat clean    → Remove everything
REM   deploy.bat backup   → Backup database
REM ============================================================

setlocal enabledelayedexpansion

echo.
echo ========================================
echo   🎭  PNJ Anonymous Bot — Deploy
echo ========================================
echo.

set "ACTION=%~1"
if "%ACTION%"=="" set "ACTION=dev"

REM Check .env file exists
if not exist "%~dp0..\.env" (
    echo ❌ .env file not found!
    echo    Copy .env.example to .env and configure it.
    exit /b 1
)

REM Check Docker is available
docker info >nul 2>&1
if errorlevel 1 (
    echo ❌ Docker is not running!
    exit /b 1
)

echo ✅ Docker is available
echo ✅ Environment file found
echo.

if "%ACTION%"=="dev" goto :dev
if "%ACTION%"=="development" goto :dev
if "%ACTION%"=="prod" goto :prod
if "%ACTION%"=="production" goto :prod
if "%ACTION%"=="stop" goto :stop
if "%ACTION%"=="logs" goto :logs
if "%ACTION%"=="log" goto :logs
if "%ACTION%"=="status" goto :status
if "%ACTION%"=="info" goto :status
if "%ACTION%"=="restart" goto :restart
if "%ACTION%"=="clean" goto :clean
if "%ACTION%"=="remove" goto :clean
if "%ACTION%"=="backup" goto :backup
goto :usage

:dev
echo 🚀 Deploying in DEVELOPMENT mode...
cd /d "%~dp0.."
docker compose up --build -d
echo.
echo ✅ Bot deployed successfully!
echo    📊 Health check: http://localhost:8080/health
echo    📋 Metrics:      http://localhost:8080/metrics
echo    📝 Logs:         docker compose logs -f pnj-bot
goto :end

:prod
echo 🚀 Deploying in PRODUCTION mode...
cd /d "%~dp0.."
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d
echo.
echo ✅ Bot deployed in production!
echo    📊 Health check: http://localhost:8080/health
echo    📋 Metrics:      http://localhost:8080/metrics
goto :end

:stop
echo 🛑 Stopping containers...
cd /d "%~dp0.."
docker compose down
echo ✅ Containers stopped
goto :end

:logs
cd /d "%~dp0.."
docker compose logs -f --tail=100 pnj-bot
goto :end

:status
echo 📊 Container Status:
docker ps -a --filter "name=pnj-anonymous-bot" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo.
echo 🏥 Health Check:
curl -s http://localhost:8080/health 2>nul || echo    ⚠️  Health endpoint not reachable
echo.
goto :end

:restart
echo 🔄 Restarting...
cd /d "%~dp0.."
docker compose restart
echo ✅ Restarted
goto :end

:clean
echo.
echo ⚠️  This will delete ALL data including the database!
set /p "CONFIRM=Are you sure? (y/N): "
if /i "%CONFIRM%"=="y" (
    cd /d "%~dp0.."
    docker compose down -v --rmi all
    echo ✅ Everything cleaned up
) else (
    echo Cancelled.
)
goto :end

:backup
echo 💾 Backing up database...
if not exist "%~dp0..\backups" mkdir "%~dp0..\backups"
set "TIMESTAMP=%date:~-4%%date:~-7,2%%date:~-10,2%_%time:~0,2%%time:~3,2%%time:~6,2%"
set "TIMESTAMP=%TIMESTAMP: =0%"
docker cp pnj-anonymous-bot:/app/data/pnj_anonymous.db "%~dp0..\backups\pnj_anonymous_%TIMESTAMP%.db"
if errorlevel 1 (
    echo ❌ Backup failed - is the container running?
) else (
    echo ✅ Backup saved to: backups\pnj_anonymous_%TIMESTAMP%.db
)
goto :end

:usage
echo Usage: %~nx0 {dev^|prod^|stop^|logs^|status^|restart^|clean^|backup}
exit /b 1

:end
endlocal
