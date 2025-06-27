-- SQL скрипт для поиска всех ocr_results для предметов 'gold coin' в категории 'buy_consumables'
-- Используется база данных: octopus

-- Основной запрос: находим все ocr_results для предметов с названием 'gold coin' и категорией 'buy_consumables'
SELECT 
    ocr.id,
    ocr.image_path,
    ocr.ocr_text,
    ocr.debug_info,
    ocr.json_data,
    ocr.raw_text,
    ocr.created_at,
    si.title,
    si.category,
    si.price,
    si.owner,
    si.count,
    si.package
FROM octopus.ocr_results ocr
INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
WHERE si.title = 'gold coin' 
  AND si.category = 'buy_consumables'
ORDER BY ocr.created_at DESC;

-- Альтернативный запрос: ID ocr_results с title для проверки
SELECT DISTINCT 
    ocr.id,
    si.title,
    si.category
FROM octopus.ocr_results ocr
INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
WHERE si.title = 'gold coin' 
  AND si.category = 'buy_consumables'
ORDER BY ocr.id;

-- Запрос для подсчета количества найденных записей с показом title
SELECT 
    si.title,
    si.category,
    COUNT(DISTINCT ocr.id) as total_ocr_results
FROM octopus.ocr_results ocr
INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
WHERE si.title = 'gold coin' 
  AND si.category = 'buy_consumables'
GROUP BY si.title, si.category;

-- Запрос для получения статистики по ценам с title
SELECT 
    si.title,
    si.category,
    si.price,
    COUNT(*) as count,
    MIN(ocr.created_at) as first_seen,
    MAX(ocr.created_at) as last_seen
FROM octopus.ocr_results ocr
INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
WHERE si.title = 'gold coin' 
  AND si.category = 'buy_consumables'
GROUP BY si.title, si.category, si.price
ORDER BY si.price;

-- НОВЫЙ ЗАПРОС: Для каждого найденного ocr_result найти все structured_items и вывести среднее из 3 минимальных стоимостей
WITH gold_coin_ocr AS (
    -- Находим все ocr_results для gold coin в buy_consumables
    SELECT DISTINCT ocr.id as ocr_id
    FROM octopus.ocr_results ocr
    INNER JOIN octopus.structured_items si ON ocr.id = si.ocr_result_id
    WHERE si.title = 'gold coin' 
      AND si.category = 'buy_consumables'
),
price_analysis AS (
    -- Для каждого ocr_result находим все structured_items и их цены
    SELECT 
        gco.ocr_id,
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
        title,
        category,
        COUNT(*) as prices_count,
        AVG(price_numeric) as avg_min_3_prices,
        MIN(price_numeric) as min_price,
        MAX(price_numeric) as max_price_of_min_3,
        GROUP_CONCAT(price ORDER BY price_numeric ASC SEPARATOR ', ') as min_3_prices,
        -- owner/count для min_price
        SUBSTRING_INDEX(GROUP_CONCAT(owner ORDER BY price_numeric ASC SEPARATOR ','), ',', 1) as min_price_owner,
        SUBSTRING_INDEX(GROUP_CONCAT(count ORDER BY price_numeric ASC SEPARATOR ','), ',', 1) as min_price_count,
        -- owner/count для max_price_of_min_3
        SUBSTRING_INDEX(SUBSTRING_INDEX(GROUP_CONCAT(owner ORDER BY price_numeric ASC SEPARATOR ','), ',', 3), ',', -1) as max_price_owner,
        SUBSTRING_INDEX(SUBSTRING_INDEX(GROUP_CONCAT(count ORDER BY price_numeric ASC SEPARATOR ','), ',', 3), ',', -1) as max_price_count
    FROM top_3_prices
    WHERE price_rank <= 3
    GROUP BY ocr_id, title, category
)
SELECT 
    am3p.title,
    am3p.category,
    am3p.avg_min_3_prices,
    am3p.min_price,
    am3p.min_price_owner,
    am3p.min_price_count,
    am3p.max_price_of_min_3,
    am3p.max_price_owner,
    am3p.max_price_count,
    ocr.created_at
FROM avg_min_3_prices am3p
INNER JOIN octopus.ocr_results ocr ON am3p.ocr_id = ocr.id
ORDER BY ocr.created_at DESC; 