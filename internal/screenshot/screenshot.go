package screenshot

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/png"
	"log"
	"octopus/internal/config"
	"os"
	"time"

	"octopus/internal/ocr"

	"github.com/kbinani/screenshot"
)

// Глобальная переменная для базы данных
var db *sql.DB

// Глобальная переменная для отслеживания необходимости сохранения скриншотов
var saveScreenshotsLocally bool

// SetDatabase устанавливает глобальную переменную базы данных
func SetDatabase(database *sql.DB) {
	db = database
}

// SetSaveScreenshotsLocally устанавливает флаг сохранения скриншотов локально
func SetSaveScreenshotsLocally(save bool) {
	saveScreenshotsLocally = save
}

// ShouldSaveLocally возвращает true, если скриншоты должны сохраняться локально
func ShouldSaveLocally() bool {
	return saveScreenshotsLocally
}

// CaptureScreenshot захватывает скриншот в память и возвращает декодированное изображение
func CaptureScreenshot(c config.CoordinatesWithSize) (image.Image, error) {
	// Определяем область для захвата с переданными координатами
	bounds := image.Rect(c.X, c.Y, c.X+c.Width, c.Y+c.Height)

	// Захватываем экран в память
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %v", err)
	}

	return img, nil
}

// CaptureFullScreen захватывает скриншот всего экрана
func CaptureFullScreen() (image.Image, error) {
	// Захватываем весь экран
	img, err := screenshot.CaptureRect(image.Rect(0, 0, 1920, 1080)) // Стандартное разрешение, можно адаптировать
	if err != nil {
		return nil, fmt.Errorf("failed to capture full screen: %v", err)
	}
	return img, nil
}

func SaveScreenshot(c config.CoordinatesWithSize, cfg *config.Config) (image.Image, error) {
	// Захватываем скриншот
	img, err := CaptureScreenshot(config.CoordinatesWithSize{X: c.X, Y: c.Y, Width: c.Width, Height: c.Height})
	if err != nil {
		log.Println("Error taking screenshot:", err)
		return nil, err
	}

	// --- Новая логика кадрирования ---
	// 40 пикселей слева, 22 пикселя сверху, 17 пикселей справа
	bounds := img.Bounds()
	cropRect := image.Rect(40, 22, bounds.Dx()-17, bounds.Dy())
	croppedImg := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(cropRect)
	// --- Конец логики кадрирования ---

	// Генерируем виртуальный путь к файлу для БД
	outputFile := fmt.Sprintf("screenshot_%d.png", time.Now().Unix())

	// Сохраняем локально только если включен флаг
	if saveScreenshotsLocally {
		// Создаем папку data, если её нет
		if err := os.MkdirAll("data", 0755); err != nil {
			log.Printf("Ошибка создания папки data: %v", err)
		} else {
			// Сохраняем изображение локально
			localFilePath := fmt.Sprintf("data/%s", outputFile)
			file, err := os.Create(localFilePath)
			if err != nil {
				log.Printf("Ошибка создания локального файла: %v", err)
			} else {
				defer file.Close()
				err = png.Encode(file, croppedImg)
				if err != nil {
					log.Printf("Ошибка сохранения локального изображения: %v", err)
				} else {
					log.Printf("📸 Скриншот сохранен локально: %s", localFilePath)
				}
			}
		}
	}

	// Создаем временный файл для OCR
	tempFile, err := os.CreateTemp("", "screenshot_*.png")
	if err != nil {
		log.Println("Error creating temp file:", err)
		return nil, err
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name()) // Удаляем временный файл
	}()

	// Сохраняем изображение во временный файл
	err = png.Encode(tempFile, croppedImg)
	if err != nil {
		log.Println("Error saving temp image:", err)
		return nil, err
	}

	// Вызываем OCR с временным файлом
	ocrResult, err := ocr.RunOCR(tempFile.Name())
	if err != nil {
		log.Printf("Ошибка OCR: %v", err)
	} else {
		fmt.Printf("OCR результат:\n%s\n", ocrResult)

		// Парсим результат OCR
		debugInfo, jsonData, rawText := ocr.ParseOCRResult(ocrResult)

		// Конвертируем изображение в байты
		imageBytes, err := imageToBytes(croppedImg)
		if err != nil {
			log.Printf("Ошибка конвертации изображения: %v", err)
		} else {
			// Сохраняем результат в базу данных
			_, err = saveOCRResultToDB(outputFile, ocrResult, debugInfo, jsonData, rawText, imageBytes, cfg)
			if err != nil {
				log.Printf("Ошибка сохранения в БД: %v", err)
			}
		}
	}

	return croppedImg, nil
}

var SaveItemOffersWithoutButtondScreenshot = func(c *config.Config) {
	SaveScreenshot(c.Screenshot.ItemOffersListWithoutButtons, c)
}

// SaveScreenshotFull захватывает и сохраняет скриншот указанной области без обрезки краёв для отладки
func SaveScreenshotFull(c config.CoordinatesWithSize) (image.Image, error) {
	// Захватываем скриншот
	img, err := CaptureScreenshot(config.CoordinatesWithSize{X: c.X, Y: c.Y, Width: c.Width, Height: c.Height})
	if err != nil {
		log.Println("Error taking screenshot:", err)
		return nil, err
	}

	fmt.Println("Full screenshot captured (not saved locally)")

	return img, nil
}

// saveOCRResultToDB сохраняет результат OCR в базу данных
func saveOCRResultToDB(imagePath, ocrResult string, debugInfo, jsonData, rawText string, imageData []byte, cfg *config.Config) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("база данных не инициализирована")
	}

	// Проверяем настройку сохранения в БД
	if cfg.SaveToDB != 1 {
		log.Printf("Сохранение в БД отключено (save_to_db = %d)", cfg.SaveToDB)
		return 0, nil
	}

	// Создаем таблицу, если она не существует
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS ocr_results (
		id INT AUTO_INCREMENT PRIMARY KEY,
		image_path VARCHAR(255) NOT NULL,
		image_data LONGBLOB,
		ocr_text LONGTEXT,
		debug_info LONGTEXT,
		json_data LONGTEXT,
		raw_text LONGTEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания таблицы: %v", err)
	}

	// Вставляем результат OCR с изображением
	insertSQL := `INSERT INTO ocr_results (image_path, image_data, ocr_text, debug_info, json_data, raw_text) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := db.Exec(insertSQL, imagePath, imageData, ocrResult, debugInfo, jsonData, rawText)
	if err != nil {
		return 0, fmt.Errorf("ошибка вставки данных: %v", err)
	}

	// Получаем ID вставленной записи
	ocrResultID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения ID записи: %v", err)
	}

	// Сохраняем структурированные данные
	if jsonData != "" {
		err = ocr.SaveStructuredData(db, int(ocrResultID), jsonData)
		if err != nil {
			log.Printf("Ошибка сохранения структурированных данных: %v", err)
		}
	}

	log.Printf("OCR результат и изображение сохранены в базу данных для файла: %s (ID: %d)", imagePath, ocrResultID)
	return int(ocrResultID), nil
}

// imageToBytes конвертирует изображение в байты в формате PNG
func imageToBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования изображения: %v", err)
	}
	return buf.Bytes(), nil
}
