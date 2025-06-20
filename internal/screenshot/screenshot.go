package screenshot

import (
	"fmt"
	"github.com/kbinani/screenshot"
	"image"
	"image/png"
	"log"
	"octopus/internal/config"
	"os"
	"path/filepath"
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

	// Сохраняем изображение
	err = png.Encode(outFile, img)
	if err != nil {
		log.Println("Error saving image:", err)
		return nil, err
	} else {
		fmt.Println("Image saved:", outputFile)
		return img, nil
	}
}

var SaveItemOffersWithoutButtondScreenshot = func(c *config.Config) {
	SaveScreenshot(c.Screenshot.ItemOffersListWithoutButtons)
}

var SaveItemOffersWithButtondScreenshot = func(c *config.Config) {
	SaveScreenshot(c.Screenshot.ItemOffersListWithButtons)
}

var SaveItemListScreenshot = func(c *config.Config) {
	SaveScreenshot(c.Screenshot.ItemList)
}
