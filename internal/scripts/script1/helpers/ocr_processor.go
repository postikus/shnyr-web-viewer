package helpers

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"octopus/internal/config"
	imageInternal "octopus/internal/image"
	"octopus/internal/ocr"
	"octopus/internal/scripts"
	"os"

	"github.com/tarm/serial"
)

// OCRProcessor содержит функции для обработки OCR
type OCRProcessor struct {
	port             *serial.Port
	config           *config.Config
	marginX          int
	marginY          int
	screenshotHelper *ScreenshotHelper
	imageHelper      *ImageHelper
	dbHelper         *DatabaseHelper
}

// NewOCRProcessor создает новый экземпляр OCRProcessor
func NewOCRProcessor(port *serial.Port, config *config.Config, marginX, marginY int, dbHelper *DatabaseHelper) *OCRProcessor {
	screenshotHelper := NewScreenshotHelper(marginX, marginY)
	imageHelper := NewImageHelper(port, config, marginX, marginY)

	return &OCRProcessor{
		port:             port,
		config:           config,
		marginX:          marginX,
		marginY:          marginY,
		screenshotHelper: screenshotHelper,
		imageHelper:      imageHelper,
		dbHelper:         dbHelper,
	}
}

// PerformScreenshotAndOCR выполняет полный цикл скриншотов и OCR
func (p *OCRProcessor) PerformScreenshotAndOCR(buttonPressed bool) error {
	counter := 0
	maxCounter := 40
	scrollRPx := 26

	// Список для хранения всех скриншотов
	var screenshots []image.Image
	var smallScreenshots []image.Image

	img := p.screenshotHelper.CaptureScreenShot()
	screenshots = append(screenshots, img)

	scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
	fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)

	// Проверяем наличие кнопок 1, 2 или 3
	button1Active := p.imageHelper.CheckButtonActive(p.config.Click.Button1.X, p.config.Click.Button1.Y, "listButton1", img)
	button2Active := p.imageHelper.CheckButtonActive(p.config.Click.Button2.X, p.config.Click.Button2.Y, "listButton2", img)
	button3Active := p.imageHelper.CheckButtonActive(p.config.Click.Button3.X, p.config.Click.Button3.Y, "listButton3", img)

	topCrop := 22 // По умолчанию обрезаем 22 пикселя сверху
	if buttonPressed || button1Active || button2Active || button3Active {
		topCrop = 45
	}

	if scrollRPx > 26 {
		scrollRPx = 26
		for counter < maxCounter && scrollRPx < 50 {
			counter, scrollRPx = p.imageHelper.CheckAndScreenScroll(counter, 1, img)
			if scrollRPx < 50 {
				img = p.screenshotHelper.CaptureScreenShot()
				screenshots = append(screenshots, img)
			}
		}

		scrollRPx = 26
		scripts.ClickCoordinates(p.port, p.config, config.Coordinates{X: p.marginX + p.config.Click.Scroll.X, Y: p.marginY + p.config.Click.Scroll.Y})
		for counter < maxCounter && scrollRPx < 50 {
			counter, scrollRPx = p.imageHelper.CheckAndClickScreenScroll(counter, img)
			if scrollRPx < 50 {
				img = p.screenshotHelper.CaptureScreenShot()
				smallScreenshots = append(smallScreenshots, img)
			}
		}

		var finalImage image.Image
		// Проверка stripe_diff между последним и предпоследним smallScreenshots
		if len(smallScreenshots) >= 2 {
			prev := smallScreenshots[len(smallScreenshots)-2]
			last := smallScreenshots[len(smallScreenshots)-1]
			// Сохраняем два последних скриншота в файлы
			f1, _ := os.Create("last_prev.png")
			defer f1.Close()
			png.Encode(f1, prev)
			f2, _ := os.Create("last.png")
			defer f2.Close()
			png.Encode(f2, last)
			diff, err := imageInternal.LastColorStripeDistanceDiff(prev, last, 26, 20)
			if err != nil {
				fmt.Printf("Ошибка stripe diff: %v\n", err)
			} else {
				finalImage, _ = imageInternal.CombineImages(screenshots, smallScreenshots[:len(smallScreenshots)-1], smallScreenshots[len(smallScreenshots)-1], diff)
				fmt.Printf("Разница stripe diff между последним и предпоследним скриншотом: %d\n", diff)
			}
		} else if len(smallScreenshots) == 1 {
			prev := screenshots[len(screenshots)-1]
			last := smallScreenshots[len(smallScreenshots)-1]
			diff, err := imageInternal.LastColorStripeDistanceDiff(prev, last, 26, 20)
			if err != nil {
				fmt.Printf("Ошибка stripe diff: %v\n", err)
			} else {
				finalImage, _ = imageInternal.CombineImages(screenshots, smallScreenshots[:len(smallScreenshots)-1], nil, 0)
				fmt.Printf("Разница stripe diff между последним и предпоследним скриншотом: %d\n", diff)
			}
		} else {
			finalImage, _ = imageInternal.CombineImages(screenshots, nil, nil, 0)
		}

		combinedImg := imageInternal.CropOpacityPixel(finalImage)
		bounds := combinedImg.Bounds()
		cropRect := image.Rect(40, topCrop, bounds.Dx()-17, bounds.Dy())
		croppedCombinedImg := combinedImg.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(cropRect)
		img = croppedCombinedImg
	} else {
		// Если скролла нет, просто кадрируем первый скриншот
		fmt.Println("⚠️ scrollRPx <= 26, делаем обычный скриншот с кадрированием")
		bounds := img.Bounds()
		cropRect := image.Rect(40, topCrop, bounds.Dx()-17, bounds.Dy())
		croppedCombinedImg := img.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(cropRect)
		img = croppedCombinedImg
	}

	fileCount, _ := CountFilesInDir("./imgs")
	fileName := fmt.Sprintf("%s/screenshot_combined_%d.png", "./imgs", fileCount)
	err := imageInternal.SaveCombinedImage(img, fileName)
	if err != nil {
		return err
	}

	scripts.ScrollUp(p.port, p.config, counter+5)

	result, err := ocr.RunOCR(fileName)
	if err != nil {
		fmt.Printf("Ошибка при выполнении OCR: %v\n", err)
		return err
	}

	// Парсим результат OCR
	debugInfo, jsonData, rawText := ocr.ParseOCRResult(result)

	// Конвертируем изображение в байты
	imageBytes, err := ImageToBytes(img)
	if err != nil {
		log.Printf("Ошибка конвертации изображения: %v", err)
		return err
	}

	// Сохраняем результат в базу данных
	_, err = p.dbHelper.SaveOCRResultToDB(fileName, result, debugInfo, jsonData, rawText, imageBytes, p.config)
	if err != nil {
		log.Printf("Ошибка сохранения в БД: %v", err)
		return err
	}

	return nil
}
