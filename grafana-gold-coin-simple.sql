-- Упрощенный Grafana SQL Query для gold coin
-- Базовые метрики для быстрого графика

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