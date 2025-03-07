@echo off
echo Docker Compose dev run
docker compose -f docker-compose-dev.yml up -d --build --force-recreate
if %errorlevel% equ 0 (
    echo success
) else (
    echo error
)
pause
