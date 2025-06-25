# Менеджеры (Managers) проекта Octopus

## LoggerManager

**Назначение:**  
Централизованное логирование в файл с поддержкой разных уровней логирования и временных меток.

**Методы:**
- `Debug(format string, args ...interface{})`  
  Записывает отладочное сообщение.
- `Info(format string, args ...interface{})`  
  Записывает информационное сообщение.
- `Error(format string, args ...interface{})`  
  Записывает сообщение об ошибке.
- `LogError(err error, context string)`  
  Записывает ошибку с дополнительной информацией.
- `Close() error`  
  Закрывает файл логов.

**Зависимости:**
- Файловая система (для записи логов)

---

## ScreenshotManager

**Назначение:**  
Управляет захватом скриншотов, поиском координат предметов на экране, обработкой изображений.

**Методы:**
- `CaptureScreenShot() image.Image`  
  Делает скриншот области с учетом отступов.
- `SaveScreenShot(cfg *config.Config) image.Image`  
  Сохраняет скриншот в файл с учетом отступов.
- `SaveScreenShotFull() image.Image`  
  Сохраняет полный скриншот без обрезки краёв.
- `GetCoordinatesItemsInItemList() ([]image.Point, error)`  
  Получает координаты всех элементов в списке предметов.
- `CheckAllButtonsStatus(img image.Image, config *config.Config, marginX, marginY int) ButtonStatus`  
  Проверяет статус всех кнопок на изображении.
- `CheckScrollExists(img image.Image) bool`  
  Проверяет наличие скролла на изображении.
- `GetPageStatus(img image.Image, config *config.Config, marginX, marginY int) PageStatus`  
  Возвращает полный статус страницы (кнопки + скролл).
- `GetScrollInfo(img image.Image) (int, int, int, error)`  
  Возвращает информацию о скролле для отладки.

**Зависимости:**
- Конфиг (`*config.Config`)
- База данных (опционально)

---

## DatabaseManager

**Назначение:**  
Инкапсулирует работу с базой данных: сохранение результатов OCR, логирование, чтение данных.

**Методы:**
- `SaveOCRResultToDB(imagePath, ocrResult, debugInfo, jsonData, rawText string, imageData []byte, cfg *config.Config) (int, error)`  
  Сохраняет результат OCR в базу данных, а также структурированные данные, если они есть.

**Зависимости:**
- SQL-драйвер (например, MySQL)
- Конфиг

---

## OCRManager

**Назначение:**  
Отвечает за распознавание текста на изображениях (OCR), обработку результатов, подготовку данных для БД.

**Методы:**
- `RunOCR(imagePath string) (string, error)`  
  Запускает внешний cpp_ocr.exe и возвращает распознанный текст.
- `fixMalformedJSON(jsonData string) string`  
  Исправляет JSON с отсутствующими запятыми в массиве structured_data (внутренний метод).
- `ParseOCRResult(ocrResult string) (debugInfo, jsonData, rawText string)`  
  Парсит результат OCR и извлекает debug-информацию, JSON и raw_text.
- `ProcessImage(img image.Image, fileName string) (result, debugInfo, jsonData, rawText string, err error)`  
  Выполняет OCR обработку изображения, парсит результат и возвращает все данные.

**Зависимости:**
- Конфиг

---

## ClickManager

**Назначение:**  
Инкапсулирует работу с Arduino и кликами по координатам, а также скроллинг и фокусировку окна.

**Методы:**
- `CheckAndScreenScroll(counter int, x int, img image.Image) (int, int)`  
  Проверяет и выполняет скролл экрана.
- `CheckAndClickScreenScroll(counter int, img image.Image) (int, int)`  
  Проверяет и кликает по скроллу.
- `saveImage(img image.Image, fileName string) error`  
  Сохраняет изображение в файл (внутренний метод).
- `combineImagesVertically(img1, img2 image.Image) (image.Image, error)`  
  Объединяет два изображения вертикально (внутренний метод).
- `PerformScreenshotWithScroll(buttonPressed bool) (image.Image, string, error)`  
  Выполняет скриншот со скроллом.
- `ClickItem(item image.Point)`  
  Кликает по элементу и обрабатывает результат.
- `ClickCoordinates(coordinate image.Point, marginX, marginY int)`  
  Выполняет клик по указанным координатам с учетом отступов.
- `FocusL2Window()`  
  Фокусирует окно L2, кликая по координатам Item1.

**Зависимости:**
- Arduino (через serial port)
- Конфиг
- ScreenshotManager
- DatabaseManager

---

## Как поддерживать документацию в актуальном состоянии

- При добавлении новых менеджеров или методов — обязательно обновляйте этот файл.
- Если меняется назначение или зависимости менеджера — отражайте это в описании.
- Для сложных методов добавляйте краткое описание логики. 