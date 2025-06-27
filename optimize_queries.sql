-- Оптимизация запросов для улучшения производительности

-- Добавляем индекс на created_at для быстрой сортировки
CREATE INDEX idx_ocr_results_created_at ON ocr_results(created_at DESC);

-- Добавляем индекс на ocr_result_id для быстрого поиска structured_items
CREATE INDEX idx_structured_items_ocr_result_id ON structured_items(ocr_result_id);

-- Добавляем составной индекс для поиска по title и category
CREATE INDEX idx_structured_items_title_category ON structured_items(title, category);

-- Добавляем индекс на price для быстрой фильтрации по цене
CREATE INDEX idx_structured_items_price ON structured_items(price);

-- Индексы для оптимизации запроса updateGoldCoinMetrics
-- Составной индекс для быстрого поиска gold coin в buy_consumables
CREATE INDEX idx_structured_items_gold_coin_buy ON structured_items(title, category, ocr_result_id) 
WHERE title = 'gold coin' AND category = 'buy_consumables';

-- Индекс для быстрого поиска по title = 'gold coin'
CREATE INDEX idx_structured_items_title_gold_coin ON structured_items(title, ocr_result_id) 
WHERE title = 'gold coin';

-- Индекс для быстрого поиска по category = 'buy_consumables'
CREATE INDEX idx_structured_items_category_buy ON structured_items(category, ocr_result_id) 
WHERE category = 'buy_consumables';

-- Индекс для быстрой фильтрации по цене (не NULL и не пустая)
CREATE INDEX idx_structured_items_price_valid ON structured_items(price, ocr_result_id) 
WHERE price IS NOT NULL AND price != '';

-- Составной индекс для JOIN между ocr_results и structured_items
CREATE INDEX idx_ocr_structured_join ON ocr_results(id, created_at);

-- Анализируем таблицы для обновления статистики
ANALYZE TABLE ocr_results;
ANALYZE TABLE structured_items; 