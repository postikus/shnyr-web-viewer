# Деплой web_viewer + Prometheus + Grafana на Render

## Состав
- **web_viewer** — Go сервис с endpoint `/metrics/gold_coin` для Prometheus
- **MySQL** — база данных
- **Prometheus** — сбор метрик с web_viewer
- **Grafana** — визуализация метрик (собранная из Dockerfile.grafana)

## Файлы для деплоя
- `Dockerfile` — для web_viewer
- `Dockerfile.grafana` — для Grafana
- `docker-compose.yml` — оркестрация всех сервисов
- `prometheus.yml` — конфигурация Prometheus

## Автоматическая настройка Grafana

### Автоматически создается:
1. **Источник данных MySQL** — подключение к БД `octopus`
2. **Дашборд "Gold Coin Analysis Dashboard"** с 3 панелями:
   - **Gold Coin - Average of 3 Min Prices** — среднее из 3 минимальных цен
   - **Gold Coin - All Prices** — все цены gold coin
   - **Gold Coin - Daily Statistics** — статистика по дням

### Конфигурационные файлы:
- `grafana/provisioning/datasources/mysql.yml` — настройка MySQL источника данных
- `grafana/provisioning/dashboards/dashboard.yml` — настройка автозагрузки дашбордов
- `grafana/dashboards/gold-coin-dashboard.json` — JSON дашборда

## Быстрый старт (локально или на Render)

1. **Склонируйте репозиторий**

2. **Проверьте структуру**
   - `Dockerfile` (для web_viewer)
   - `Dockerfile.grafana` (для Grafana)
   - `docker-compose.yml`
   - `prometheus.yml`
   - `grafana/` папка с конфигурациями

3. **Запустите всё через Docker Compose**

```sh
docker-compose up --build
```

4. **Проверьте сервисы**
   - web_viewer: http://localhost:8080
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (логин: admin/admin)

5. **Дашборд готов!**
   - Автоматически создан источник данных MySQL
   - Автоматически создан дашборд "Gold Coin Analysis Dashboard"
   - Все панели настроены и готовы к работе

## Структура дашборда

### Панель 1: Average of 3 Min Prices
- Показывает среднее из 3 минимальных цен для каждого OCR результата
- Использует сложный SQL запрос с CTE
- Поле: `value` (среднее значение)

### Панель 2: All Prices
- Показывает все цены gold coin
- Простой график всех найденных цен
- Поле: `price_value`

### Панель 3: Daily Statistics
- Статистика по дням: количество OCR, количество предметов, средние/мин/макс цены
- Группировка по дням
- Поля: `ocr_count`, `items_count`, `avg_price`, `min_price`, `max_price`

## Деплой на Render

1. **Создайте новый Blueprint (YAML) проект на Render**
2. Загрузите файлы в корень репозитория:
   - `Dockerfile` (для web_viewer)
   - `Dockerfile.grafana` (для Grafana)
   - `docker-compose.yml`
   - `prometheus.yml`
   - `grafana/` папка с конфигурациями
3. Render автоматически поднимет все сервисы
4. Откройте Grafana по публичному адресу Render
5. **Дашборд уже создан и готов к работе!**

## Структура Dockerfile.grafana

- Базовый образ: `grafana/grafana:latest`
- Порт: 3000
- Логин по умолчанию: admin/admin
- Автоматическая настройка источников данных и дашбордов
- Копирование всех конфигураций при сборке

## Дополнительные возможности

### Ручное добавление панелей
Если нужно добавить новые панели:
1. Откройте дашборд в Grafana
2. Добавьте новую панель
3. Используйте SQL запросы из `grafana-gold-coin-query.sql` или `grafana-gold-coin-simple.sql`

### Изменение дашборда
1. Отредактируйте `grafana/dashboards/gold-coin-dashboard.json`
2. Пересоберите контейнер Grafana
3. Дашборд обновится автоматически

---

**Внимание:**
- Для production рекомендуется использовать отдельный volume для MySQL
- Не забудьте сменить пароли по умолчанию
- Для доступа к Grafana снаружи настройте публичный порт 3000

---

**Авторы:**
- web_viewer: ваш Go сервис
- Grafana: собранная из Dockerfile.grafana с автоматической настройкой
- Prometheus: официальный образ 