# SQL Запросы для Grafana

## Файлы
- `grafana-gold-coin-query.sql` - Полный запрос с анализом 3 минимальных цен
- `grafana-gold-coin-simple.sql` - Упрощенный запрос для быстрого графика

## Настройка в Grafana

### 1. Добавление источника данных MySQL
1. Перейдите в **Configuration** → **Data Sources**
2. Нажмите **Add data source**
3. Выберите **MySQL**
4. Настройте подключение к вашей БД `octopus`

### 2. Создание дашборда
1. Создайте новый дашборд
2. Добавьте панель типа **Time series**

### 3. Использование запросов

#### Вариант 1: Упрощенный график цен
```sql
-- Вставьте содержимое grafana-gold-coin-simple.sql
SELECT 
    ocr.created_at as time,
    ocr.id as ocr_id,
    si.title,
    si.category,
    CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2)) as price_value
FROM octopus.ocr_results ocr
INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
WHERE si.title = 'gold coin' 
  AND si.category = 'buy_consumables'
  AND si.price IS NOT NULL 
  AND si.price != ''
  AND CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2)) > 0
ORDER BY ocr.created_at ASC;
```

**Настройки панели:**
- **Field**: `price_value`
- **Time**: `time`
- **Legend**: `ocr_id`

#### Вариант 2: Анализ средних цен
```sql
-- Вставьте содержимое grafana-gold-coin-query.sql
-- (полный запрос с CTE)
```

**Настройки панели:**
- **Field**: `value` (среднее из 3 минимальных цен)
- **Time**: `time`
- **Legend**: `ocr_id`

### 4. Дополнительные панели

#### Статистика по ценам
```sql
SELECT 
    ocr.created_at as time,
    COUNT(*) as items_count,
    AVG(CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2))) as avg_price,
    MIN(CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2))) as min_price,
    MAX(CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2))) as max_price
FROM octopus.ocr_results ocr
INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
WHERE si.title = 'gold coin' 
  AND si.category = 'buy_consumables'
  AND si.price IS NOT NULL 
  AND si.price != ''
GROUP BY DATE(ocr.created_at)
ORDER BY ocr.created_at ASC;
```

#### Количество найденных предметов по дням
```sql
SELECT 
    DATE(ocr.created_at) as time,
    COUNT(DISTINCT ocr.id) as ocr_count,
    COUNT(si.id) as items_count
FROM octopus.ocr_results ocr
INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
WHERE si.title = 'gold coin' 
  AND si.category = 'buy_consumables'
GROUP BY DATE(ocr.created_at)
ORDER BY time ASC;
```

## Рекомендуемые дашборды

### 1. Обзор цен
- График всех цен gold coin
- Средние цены по дням
- Минимальные/максимальные цены

### 2. Анализ трендов
- Среднее из 3 минимальных цен
- Количество найденных предметов
- Статистика по владельцам

### 3. Мониторинг активности
- Количество OCR результатов по дням
- Время между находками
- Активность по часам

## Примечания

- Все запросы используют `created_at as time` для временных меток
- Цены преобразуются из строки в DECIMAL для корректного отображения
- Фильтры по `title = 'gold coin'` и `category = 'buy_consumables'` можно изменить
- Для больших объемов данных рекомендуется добавить индексы на `created_at` 