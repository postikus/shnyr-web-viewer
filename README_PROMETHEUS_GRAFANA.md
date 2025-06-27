# Деплой web_viewer + Prometheus + Grafana на Render

## Состав
- **web_viewer** — Go сервис с endpoint `/metrics/gold_coin` для Prometheus
- **MySQL** — база данных
- **Prometheus** — сбор метрик с web_viewer
- **Grafana** — визуализация метрик

## Быстрый старт (локально или на Render)

1. **Склонируйте репозиторий**

2. **Проверьте структуру**
   - `Dockerfile` (для web_viewer)
   - `docker-compose.yml`
   - `prometheus.yml`

3. **Запустите всё через Docker Compose**

```sh
docker-compose up --build
```

4. **Проверьте сервисы**
   - web_viewer: http://localhost:8080
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (логин: admin/admin)

5. **Добавьте Prometheus как источник данных в Grafana**
   - URL: `http://prometheus:9090`

6. **Создайте дашборд в Grafana**
   - Метрики: `gold_coin_price`, `gold_coin_min_price`, `gold_coin_max_price`, `gold_coin_timestamp`
   - Используйте label `ocr_id` или `title` для фильтрации

## Пример запроса для графика в Grafana

```
gold_coin_price
```

или

```
gold_coin_min_price
```

или

```
gold_coin_max_price
```

## Деплой на Render

1. **Создайте новый Blueprint (YAML) проект на Render**
2. Загрузите файлы `docker-compose.yml`, `prometheus.yml`, `Dockerfile` в корень репозитория
3. Render автоматически поднимет все сервисы
4. Откройте Grafana по публичному адресу Render

---

**Внимание:**
- Для production рекомендуется использовать отдельный volume для MySQL
- Не забудьте сменить пароли по умолчанию
- Для доступа к Grafana снаружи настройте публичный порт 3000

---

**Авторы:**
- web_viewer: ваш Go сервис
- Prometheus/Grafana: официальные образы 