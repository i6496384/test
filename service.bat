@echo off
echo ================================
echo WireGuard Web Manager - Service
echo ================================
echo.

REM Проверяем наличие исполняемого файла
if not exist "wireguard-web-manager.exe" (
    echo ОШИБКА: файл wireguard-web-manager.exe не найден
    echo Запустите сначала build-and-run.bat для сборки
    pause
    exit /b 1
)

echo Доступные команды:
echo   install  - Установить как Windows сервис
echo   uninstall - Удалить Windows сервис
echo   start    - Запустить сервис
echo   stop     - Остановить сервис
echo.

if "%1"=="install" goto INSTALL
if "%1"=="uninstall" goto UNINSTALL
if "%1"=="start" goto START
if "%1"=="stop" goto STOP

echo Использование: service.bat [install^|uninstall^|start^|stop]
pause
exit /b 1

:INSTALL
echo Установка WireGuard Web Manager как Windows сервис...
sc create "WireGuardWebManager" binPath= "%CD%\wireguard-web-manager.exe" DisplayName= "WireGuard Web Manager"
if errorlevel 1 (
    echo Ошибка при установке сервиса
    pause
    exit /b 1
)
echo Сервис установлен успешно
echo Для запуска используйте: service.bat start
pause
exit /b 0

:UNINSTALL
echo Удаление WireGuard Web Manager сервиса...
sc stop "WireGuardWebManager" >nul 2>&1
sc delete "WireGuardWebManager"
echo Сервис удален
pause
exit /b 0

:START
echo Запуск WireGuard Web Manager сервиса...
sc start "WireGuardWebManager"
pause
exit /b 0

:STOP
echo Остановка WireGuard Web Manager сервиса...
sc stop "WireGuardWebManager"
pause
exit /b 0