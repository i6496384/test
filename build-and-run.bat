@echo off
echo ================================
echo WireGuard Web Manager - Windows
echo ================================
echo.

REM Проверяем наличие Go
go version >nul 2>&1
if errorlevel 1 (
    echo ОШИБКА: Go не установлен или не найден в PATH
    echo Скачайте Go с https://golang.org/dl/
    pause
    exit /b 1
)

echo Найден Go: 
go version

REM Скачиваем зависимости
echo.
echo Скачивание зависимостей...
go mod download

REM Собираем приложение
echo.
echo Сборка приложения...
go build -o wireguard-web-manager.exe

if errorlevel 1 (
    echo ОШИБКА при сборке приложения
    pause
    exit /b 1
)

echo.
echo Сборка завершена успешно!
echo.

REM Запускаем приложение
echo Запуск WireGuard Web Manager...
echo.
echo Веб-интерфейс будет доступен по адресу: http://localhost:8080
echo Для остановки нажмите Ctrl+C
echo.

wireguard-web-manager.exe