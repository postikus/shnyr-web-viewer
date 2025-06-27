-- Оптимизация запросов для улучшения производительности

-- Добавляем индекс на created_at для быстрой сортировки
CREATE INDEX idx_ocr_results_created_at ON ocr_results(created_at DESC);

-- Добавляем индекс на ocr_result_id для быстрого поиска structured_items
CREATE INDEX idx_structured_items_ocr_result_id ON structured_items(ocr_result_id);

-- Добавляем составной индекс для поиска по title и category
CREATE INDEX idx_structured_items_title_category ON structured_items(title, category);

-- Добавляем индекс на price для быстрой фильтрации по цене
CREATE INDEX idx_structured_items_price ON structured_items(price);

-- Анализируем таблицы для обновления статистики
ANALYZE TABLE ocr_results;
ANALYZE TABLE structured_items; 