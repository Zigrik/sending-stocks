# Sending Stocks - Мультибрендовый сервис отправки остатков

Сервис для загрузки, обработки и отправки файлов остатков из 1С в системы производителей шин (Pirelli, Ikon, Cordiant и др.).

## Возможности

- 📤 Загрузка XLSX файлов из 1С (ведомость остатков)
- 🔍 Автоматическое распознавание брендов и сезонности
- 📊 Формирование отчетов для разных производителей:
  - **Pirelli** - CSV для API + Excel отчет (все позиции)
  - **Ikon** - сводный Excel отчет с группировкой по сезонам
  - **Cordiant** - CSV для API + фильтрация по коду производителя
- 📧 Отправка отчетов по email (SMTP)
- 🔒 Защита паролем
- 🎨 Удобный веб-интерфейс с toast-уведомлениями

## Установка

```bash
# Клонирование репозитория
git clone https://github.com/yourusername/sending-stocks.git
cd sending-stocks

# Установка зависимостей
go mod tidy

# Сборка
go build -o stock-server

# Запуск
./stock-server

Конфигурация
Создайте файл .env в корневой директории на основе примера ниже:

env
# Сервер
SERVER_PORT=8080
ADMIN_PASSWORD=your_secure_password

# Директории
UPLOAD_DIR=./uploads
PROCESSED_DIR=./uploads/processed

# SMTP Configuration (для отправки email)
SMTP_HOST=smtp.mail.ru
SMTP_PORT=587
SMTP_USERNAME=your_email@mail.ru
SMTP_PASSWORD=your_password
SMTP_FROM=your_email@mail.ru

# Email recipients по умолчанию (можно несколько через запятую)
PIRELLI_EMAILS=manager@company.ru,report@company.ru
IKON_EMAILS=ikon@company.ru
CORDIANT_EMAILS=cordiant@company.ru

# Pirelli
PIRELLI_BRANDS=Pirelli,Formula
PIRELLI_BASE_URL=https://reports.pirelli.ru/local/templates/dealer/ajax/api.php
PIRELLI_LOGIN=5700097
PIRELLI_TOKEN=your_pirelli_token
PIRELLI_CUSTOMER_CODE=5700097

# Ikon
IKON_COMPANY_NAME=IP SEMISOTNOV
IKON_SUMMER_A=Ikon Autograph,Nokian Hakka
IKON_SUMMER_B=Ikon Character,Nordman by Nokian
IKON_SUMMER_C=Bars
IKON_SUMMER_D=Attar
IKON_WINTER_A=Ikon Autograph,Nokian
IKON_WINTER_B=Ikon Character,Nordman by Nokian
IKON_WINTER_C=Attar

# Cordiant
CORDIANT_BRANDS=Cordiant,Gislaved,Torero,Tunga
CORDIANT_BASE_URL=https://b2b.cordiant.ru/rest/
CORDIANT_TOKEN=your_cordiant_token
CORDIANT_LOGIN=your_login
CORDIANT_PASSWORD=your_password
Использование
Запустите сервер: ./stock-server

Откройте браузер по адресу: http://localhost:8080

Введите пароль администратора (из .env)

Загрузите XLSX файл из 1С (ведомость остатков)

Дождитесь обработки файла

Выберите нужный отчет:

Pirelli: скачать CSV (с SKU), отправить в API, скачать Excel, отправить по email

Ikon: скачать Excel, отправить по email

Cordiant: скачать CSV, отправить в API

API Endpoints
Метод	Эндпоинт	Описание
POST	/api/check-password	Проверка пароля
POST	/api/upload	Загрузка XLSX файла
POST	/api/process	Обработка файла
GET	/api/download-pirelli-csv	Скачать CSV для Pirelli
POST	/api/send-pirelli	Отправить в Pirelli API
GET	/api/download-pirelli-excel	Скачать Excel отчет Pirelli
POST	/api/send-pirelli-excel	Отправить Pirelli по email
GET	/api/download-ikon	Скачать Excel отчет Ikon
POST	/api/send-ikon	Отправить Ikon по email
GET	/api/download-cordiant-csv	Скачать CSV для Cordiant
POST	/api/send-cordiant	Отправить в Cordiant API
POST	/api/clear	Очистить загруженные файлы
Структура проекта
text
sending-stocks/
├── main.go                 # Точка входа
├── handlers/               # HTTP обработчики
│   ├── web.go              # Веб-интерфейс
│   └── upload.go           # Обработка загрузок
├── processors/             # Обработчики брендов
│   ├── parser.go           # Парсер XLSX
│   ├── pirelli.go          # Pirelli CSV
│   ├── pirelli_excel.go    # Pirelli Excel
│   ├── ikon.go             # Ikon отчет
│   └── cordiant.go         # Cordiant отчет
├── services/               # Внешние сервисы
│   ├── pirelli_api.go      # Pirelli API
│   ├── cordiant_api.go     # Cordiant API
│   └── smtp.go             # Email отправка
├── models/                 # Модели данных
│   └── models.go
├── templates/              # HTML шаблоны
│   └── form.html
├── uploads/                # Загруженные файлы
└── .env                    # Конфигурация
Формат входного файла
Ожидается XLSX файл из 1С со следующей структурой (начиная с 12 строки):

Колонка	Поле	Описание
A	Наименование	Полное наименование товара
C	Бренд + сезон	Например: "Pirelli лето" или "Cordiant зима"
F	Код 1С	Внутренний код товара
G	Код производителя	CAI или артикул производителя
H	Типоразмер	Размер шины
I	Остаток	Количество на складе
J	Цена	Цена (формат "1 234,56")
Требования
Go 1.24 или выше

Доступ к SMTP серверу для отправки email

API ключи для Pirelli и Cordiant (опционально)