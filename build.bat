@echo off
setlocal EnableDelayedExpansion

:: ============================================
:: ProxiFyre Manager - Build Script for Windows
:: ============================================

:: Переключаем консоль на UTF-8 для корректного отображения символов
chcp 65001 >nul 2>&1

:: Настройки
set APP_NAME=ProxiFyreManager
set OUTPUT_DIR=dist
set ICON_FILE=ProxiFyre.png
set MAIN_FILE=main.go

:: Символы: если консоль не поддерживает UTF-8, замените на [+], [!], [>]
set "CHECK=✓"
set "CROSS=✗"
set "INFO=→"

:: Проверка наличия Go
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo %CROSS% Go is not installed or not in PATH
    pause
    exit /b 1
)
echo %CHECK% Go found

:: Проверка основного файла
if not exist "%MAIN_FILE%" (
    echo %CROSS% %MAIN_FILE% not found in current directory
    pause
    exit /b 1
)
echo %CHECK% Found: %MAIN_FILE%

:: Проверка иконки (опционально)
if exist "%ICON_FILE%" (
    echo %CHECK% Icon found: %ICON_FILE%
) else (
    echo %INFO% Icon not found: %ICON_FILE% ^(app will use default^)
)

:: Очистка и создание папки вывода
echo.
echo %INFO% Preparing output directory...
if exist "%OUTPUT_DIR%" rmdir /s /q "%OUTPUT_DIR%"
mkdir "%OUTPUT_DIR%"
mkdir "%OUTPUT_DIR%\configs"

:: Сборка приложения
echo %INFO% Building %APP_NAME%.exe (GUI mode, no console)...
go build -ldflags "-H windowsgui -s -w" -o "%OUTPUT_DIR%\%APP_NAME%.exe" .

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo %CROSS% Build failed!
    echo    Check Go installation and dependencies.
    pause
    exit /b 1
)
echo %CHECK% Build successful: %OUTPUT_DIR%\%APP_NAME%.exe

:: Копирование ресурсов
if exist "%ICON_FILE%" (
    copy "%ICON_FILE%" "%OUTPUT_DIR%\" >nul
    echo %CHECK% Copied: %ICON_FILE%
)

:: Создание дефолтного конфига
if not exist "%OUTPUT_DIR%\configs\default.json" (
    echo {"logLevel":"Error","proxies":[],"excludes":[]} > "%OUTPUT_DIR%\configs\default.json"
    echo %CHECK% Created: configs\default.json
)

:: Файл с версией и датой (через PowerShell вместо устаревшего wmic)
for /f "tokens=2 delims==" %%a in ('powershell -NoProfile -Command "Get-Date -Format 'yyyy-MM-dd'"') do set "BUILD_DATE=%%a"
echo ProxiFyre Manager v2.0.0.1 %BUILD_DATE% > "%OUTPUT_DIR%\version.txt"
echo %CHECK% Created: version.txt

:: README для дистрибутива
(
    echo ProxiFyre Configuration Manager
    echo ================================
    echo.
    echo Запустите от имени администратора для установки/удаления службы.
    echo.
    echo Файлы:
    echo - %APP_NAME%.exe      : Основное приложение
    echo - configs/            : Папка с конфигурациями
    echo - %ICON_FILE%         : Иконка приложения ^(опционально^)
    echo.
    echo Горячие клавиши:
    echo - Ctrl+Q : Выход
    echo.
    echo Поддержка: rogverse.fyi
) > "%OUTPUT_DIR%\README.txt"
echo %CHECK% Created: README.txt

:: Итог
echo.
echo ========================================
echo %CHECK% Build complete!
echo %INFO% Output: %OUTPUT_DIR%\
echo ========================================
echo.
dir "%OUTPUT_DIR%" /b
echo.
pause