# Система мониторинга цен Gold Coin

## Описание

Система мониторинга для отслеживания цен на Gold Coin в игре. Включает:

- **Web Viewer** (порт 8080) - веб-интерфейс с метриками Prometheus
- **Prometheus** (порт 9090) - сбор и хранение метрик
- **Grafana** (порт 3000) - визуализация данных

## Метрики

Система отслеживает следующие метрики для Gold Coin:

- `gold_coin_avg_min_3_prices` - среднее из 3 минимальных цен
- `gold_coin_min_price` - минимальная цена
- `gold_coin_max_price_of_min_3` - максимальная из 3 минимальных цен
- `gold_coin_prices_count` - количество найденных цен

## Запуск

### 1. Сборка и запуск всех сервисов

```bash
docker-compose up --build
```

### 2. Доступ к сервисам

- **Web Viewer**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

### 3. Настройка Grafana

1. Войдите в Grafana (admin/admin)
2. Dashboard "Gold Coin Prices Monitoring" должен быть автоматически загружен
3. Если dashboard не появился, импортируйте файл `grafana/dashboards/gold-coin-prices.json`

## Архитектура

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Web Viewer  │───▶│ Prometheus  │───▶│   Grafana   │
│   :8080     │    │   :9090     │    │   :3000     │
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   /metrics  │    │   Storage   │    │  Dashboard  │
│  endpoint   │    │   (TSDB)    │    │   Graphs    │
└─────────────┘    └─────────────┘    └─────────────┘
```

## Обновление метрик

- Web Viewer обновляет метрики каждые 30 секунд
- Prometheus собирает метрики каждые 30 секунд
- Grafana обновляет графики каждые 30 секунд

## Структура файлов

```
├── cmd/web_viewer/main.go          # Web сервер с метриками
├── prometheus.yml                  # Конфигурация Prometheus
├── docker-compose.yml              # Оркестрация сервисов
├── Dockerfile.web-viewer           # Сборка web_viewer
├── grafana/
│   ├── datasources/prometheus.yml  # Источник данных
│   └── dashboards/gold-coin-prices.json # Dashboard
└── find_gold_coin_ocr.sql          # SQL запрос для анализа
```

## Мониторинг

### Prometheus Queries

```promql
# Средняя цена
gold_coin_avg_min_3_prices

# Минимальная цена
gold_coin_min_price

# Максимальная из 3 минимальных
gold_coin_max_price_of_min_3

# Количество цен
gold_coin_prices_count
```

### Grafana Dashboard

Dashboard содержит 4 панели:
1. **Average Min 3 Prices** - график средних цен
2. **Min Price** - график минимальных цен
3. **Max Price of Min 3** - график максимальных из 3 минимальных
4. **Prices Count** - количество найденных цен

## Остановка

```bash
docker-compose down
```

Для удаления данных:
```bash
docker-compose down -v
``` 