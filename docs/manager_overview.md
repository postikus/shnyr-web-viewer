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
  Делает скриншот области, соответствующей окну предметов.
- `SaveScreenShot(cfg *config.Config) image.Image`  
  Сохраняет скриншот области в файл, используя параметры из конфига.
- `SaveScreenShotFull() image.Image`  
  Сохраняет полный скриншот области без обрезки краёв для отладки.
- `GetCoordinatesItemsInItemList(img image.Image) ([]image.Point, error)`  
  Возвращает массив точек для клика по предметам на переданном изображении.
- `GetItemListItemsCoordinates() ([]image.Point, error)`  
  Делает скриншот и возвращает координаты всех предметов на странице.
- `findItemPositionsByTextColor(img image.Image, targetX int) []image.Point`  
  Находит центры цветных строк с названиями предметов (внутренний метод).

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
  Проверяет и выполняет скролл экрана, если это необходимо.
- `CheckAndClickScreenScroll(counter int, img image.Image) (int, int)`  
  Проверяет и кликает по скроллу, если это необходимо.
- `saveImage(img image.Image, fileName string) error`  
  Сохраняет изображение в файл (внутренний метод).
- `combineImagesVertically(img1, img2 image.Image) (image.Image, error)`  
  Объединяет два изображения вертикально (внутренний метод).
- `PerformScreenshotWithScroll(buttonPressed bool) (image.Image, string, error)`  
  Делает скриншот с прокруткой, объединяет изображения, сохраняет результат.
- `ClickItem(item config.Coordinates)`  
  Кликает по элементу (заготовка для расширения).
- `FocusL2Window()`  
  Фокусирует окно L2, кликая по координатам Item1.
- `ClickCoordinates(coordinates config.Coordinates)`  
  Выполняет клик по указанным координатам.

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