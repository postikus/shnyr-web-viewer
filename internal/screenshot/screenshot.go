package screenshot

import (
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

func SaveScreenshot(c config.CoordinatesWithSize) (image.Image, error) {
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
		}

		return croppedImg, nil
	}
}

var SaveItemOffersWithoutButtondScreenshot = func(c *config.Config) {
	SaveScreenshot(c.Screenshot.ItemOffersListWithoutButtons)
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
