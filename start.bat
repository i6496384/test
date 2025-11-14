@echo off
echo ================================
echo WireGuard Web Manager - Запуск
echo ================================
echo.

REM Проверяем наличие исполняемого файла
if not exist "wireguard-web-manager.exe" (
    echo ОШИБКА: файл wireguard-web-manager.exe не найден
    echo Запустите сначала build-and-run.bat для сборки
    pause
    exit /b 1
)

echo Найден исполняемый файл: wireguard-web-manager.exe
echo.

REM Запускаем приложение
echo Запуск WireGuard Web Manager...
echo.
echo Веб-интерфейс будет доступен по адресу: http://localhost:8080
echo Для остановки нажмите Ctrl+C
echo.

wireguard-web-manager.exe