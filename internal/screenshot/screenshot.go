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
	"path/filepath"

	"octopus/internal/ocr"

	"github.com/kbinani/screenshot"
)

// Глобальная переменная для базы данных
var db *sql.DB

// SetDatabase устанавливает глобальную переменную базы данных
func SetDatabase(database *sql.DB) {
	db = database
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
	// Получаем список файлов в папке ./imgs/
	files, err := filepath.Glob("./imgs/*")
	if err != nil {
		log.Println("Error reading files in ./imgs/:", err)
		return nil, err
	}

	// Количество файлов в папке
	screenshotCount := len(files)

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

	// Генерируем имя файла с номером, основанным на количестве файлов
	outputFile := fmt.Sprintf("./imgs/screenshot%d.png", screenshotCount+1)

	// Создаем файл для сохранения
	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Println("Error creating file:", err)
		return nil, err
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {

		}
	}(outFile)

	// Сохраняем обрезанное изображение
	err = png.Encode(outFile, croppedImg)
	if err != nil {
		log.Println("Error saving image:", err)
		return nil, err
	} else {
		fmt.Println("Image saved:", outputFile)

		// Вызываем OCR напрямую
		ocrResult, err := ocr.RunOCR(outputFile)
		if err != nil {
			log.Printf("Ошибка OCR: %v", err)
		} else {
			fmt.Printf("OCR результат:\n%s\n", ocrResult)

			// Парсим результат OCR
			debugInfo, jsonData := ocr.ParseOCRResult(ocrResult)

			// Конвертируем изображение в байты
			imageBytes, err := imageToBytes(croppedImg)
			if err != nil {
				log.Printf("Ошибка конвертации изображения: %v", err)
			} else {
				// Сохраняем результат в базу данных
				err = saveOCRResultToDB(outputFile, ocrResult, debugInfo, jsonData, imageBytes, cfg)
				if err != nil {
					log.Printf("Ошибка сохранения в БД: %v", err)
				}
			}
		}

		return croppedImg, nil
	}
}

var SaveItemOffersWithoutButtondScreenshot = func(c *config.Config) {
	SaveScreenshot(c.Screenshot.ItemOffersListWithoutButtons, c)
}

// SaveScreenshotFull захватывает и сохраняет скриншот указанной области без обрезки краёв для отладки
func SaveScreenshotFull(c config.CoordinatesWithSize) (image.Image, error) {
	// Получаем список файлов в папке ./imgs/
	files, err := filepath.Glob("./imgs/*")
	if err != nil {
		log.Println("Error reading files in ./imgs/:", err)
		return nil, err
	}

	// Количество файлов в папке
	screenshotCount := len(files)

	// Захватываем скриншот
	img, err := CaptureScreenshot(config.CoordinatesWithSize{X: c.X, Y: c.Y, Width: c.Width, Height: c.Height})
	if err != nil {
		log.Println("Error taking screenshot:", err)
		return nil, err
	}

	// Генерируем имя файла с номером, основанным на количестве файлов
	outputFile := fmt.Sprintf("./imgs/full_screenshot%d.png", screenshotCount+1)

	// Создаем файл для сохранения
	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Println("Error creating file:", err)
		return nil, err
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {

		}
	}(outFile)

	// Сохраняем изображение без обрезки краёв
	err = png.Encode(outFile, img)
	if err != nil {
		log.Println("Error saving image:", err)
		return nil, err
	} else {
		fmt.Println("Full screenshot saved:", outputFile)

		// TODO: Добавить прямую интеграцию OCR здесь
		// Вместо вызова exec.Command

		return img, nil
	}
}

// saveOCRResultToDB сохраняет результат OCR в базу данных
func saveOCRResultToDB(imagePath, ocrResult string, debugInfo, jsonData string, imageData []byte, config *config.Config) error {
	if db == nil {
		return fmt.Errorf("база данных не инициализирована")
	}

	// Проверяем настройку сохранения в БД
	if config.SaveToDB != 1 {
		log.Printf("Сохранение в БД отключено (save_to_db = %d)", config.SaveToDB)
		return nil
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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы: %v", err)
	}

	// Вставляем результат OCR с изображением
	insertSQL := `INSERT INTO ocr_results (image_path, image_data, ocr_text, debug_info, json_data) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(insertSQL, imagePath, imageData, ocrResult, debugInfo, jsonData)
	if err != nil {
		return fmt.Errorf("ошибка вставки данных: %v", err)
	}

	log.Printf("OCR результат и изображение сохранены в базу данных для файла: %s", imagePath)
	return nil
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
