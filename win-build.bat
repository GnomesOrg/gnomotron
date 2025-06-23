@echo off
echo Docker Compose run
docker compose up -d --build --force-recreate
if %errorlevel% equ 0 (
    echo success
) else (
    echo error
)
