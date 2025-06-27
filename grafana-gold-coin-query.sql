-- Grafana SQL Query для gold coin метрик
-- Используется для создания временных рядов в Grafana

WITH gold_coin_ocr AS (
    -- Находим все ocr_results для gold coin в buy_consumables
    SELECT DISTINCT ocr.id as ocr_id, ocr.created_at
    FROM octopus.ocr_results ocr
    INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
    WHERE si.title = 'gold coin' 
      AND si.category = 'buy_consumables'
),
price_analysis AS (
    -- Для каждого ocr_result находим все structured_items и их цены
    SELECT 
        gco.ocr_id,
        gco.created_at,
        si.id as structured_item_id,
        si.title,
        si.category,
        si.price,
        si.owner,
        si.count,
        si.package,
        -- Преобразуем цену в числовое значение для сортировки
        CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2)) as price_numeric
    FROM gold_coin_ocr gco
    INNER JOIN octopus.structured_items si ON gco.ocr_id = si.ocr_result_id
    WHERE si.price IS NOT NULL 
      AND si.price != ''
      AND CAST(REPLACE(REPLACE(si.price, ',', ''), ' ', '') AS DECIMAL(15,2)) > 0
),
top_3_prices AS (
    -- Выбираем 3 минимальные цены для каждого ocr_result
    SELECT 
        ocr_id,
        created_at,
        title,
        category,
        price,
        price_numeric,
        owner,
        count,
        package,
        ROW_NUMBER() OVER (PARTITION BY ocr_id ORDER BY price_numeric ASC) as price_rank
    FROM price_analysis
),
avg_min_3_prices AS (
    -- Вычисляем среднее из 3 минимальных цен для каждого ocr_result
    SELECT 
        ocr_id,
        created_at,
        title,
        category,
        COUNT(*) as prices_count,
        AVG(price_numeric) as avg_min_3_prices,
        MIN(price_numeric) as min_price,
        MAX(price_numeric) as max_price_of_min_3,
        GROUP_CONCAT(price ORDER BY price_numeric ASC SEPARATOR ', ') as min_3_prices
    FROM top_3_prices
    WHERE price_rank <= 3
    GROUP BY ocr_id, created_at, title, category
)
-- Финальный результат для Grafana
SELECT 
    created_at as time,  -- Временная метка для Grafana
    ocr_id,
    title,
    category,
    prices_count,
    avg_min_3_prices as value,  -- Основное значение для графика
    min_price,
    max_price_of_min_3,
    min_3_prices
FROM avg_min_3_prices
ORDER BY created_at ASC;  -- Сортировка по времени для временного ряда 