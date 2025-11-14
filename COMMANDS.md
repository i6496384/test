# Команды для запуска WireGuard Web Manager

## Быстрый запуск

```bash
# Перейдите в папку проекта
cd wireguard-web-manager

# Установите зависимости
go mod tidy

# Запустите приложение
go run main.go
```

## Проверка установки Go

```bash
go version
```

## Дополнительные команды

### Сборка проекта
```bash
go build -o wireguard-manager main.go
```

### Запуск собранного файла
```bash
./wireguard-manager
```

### Очистка модулей
```bash
go clean -modcache
```

## Проверка порта

После запуска приложение будет доступно по адресу:
http://localhost:8080

## Структура папок проекта

```
wireguard-web-manager/
├── go.mod                    # Зависимости Go
├── main.go                   # Основной файл
├── README.md                 # Документация
├── COMMANDS.md               # Этот файл
├── models/                   # Модели данных
│   └── models.go
├── handlers/                 # Обработчики
│   └── handlers.go
├── templates/                # HTML шаблоны
│   ├── base.html
│   ├── index.html
│   └── dashboard.html
├── static/                   # Статические файлы
│   └── app.js
└── uploads/                  # Папка для загрузок (создается автоматически)
```