package scpript1

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"octopus/internal/config"
	imageInternal "octopus/internal/image"
	"octopus/internal/ocr"
	"octopus/internal/screen"
	"octopus/internal/screenshot"
	"octopus/internal/scripts"

	"github.com/tarm/serial"
)

var Run = func(port *serial.Port, c *config.Config, db *sql.DB) {

	// Количество пикселей для отрезания сверху
	topOffset := 23

	// Делаем скриншот всего экрана
	img, err := screen.CaptureFullScreen()
	if err != nil {
		log.Fatalf("Ошибка захвата экрана: %v", err)
	}

	// Обрезаем верхние пиксели
	bounds := img.Bounds()
	croppedImg := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()-topOffset))
	for y := topOffset; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			croppedImg.Set(x, y-topOffset, img.At(x, y))
		}
	}

	// Ищем окно
	gameWindow, err := imageInternal.FindGameWindow(croppedImg)
	if err != nil {
		fmt.Printf("Окно не найдено: %v\n", err)
		return
	}

	marginX := gameWindow.X - 150
	marginY := gameWindow.Y + topOffset + 48

	fmt.Printf("marginX, marginY: %v %v\n", marginX, marginY)

	var captureScreenShot = func() image.Image {
		img, _ := screenshot.CaptureScreenshot(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361})
		return img
	}

	var saveScreenShot = func() image.Image {
		img, _ := screenshot.SaveScreenshot(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361}, c)
		return img
	}

	var _ = func() image.Image {
		img, _ := screenshot.SaveScreenshotFull(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361})
		return img
	}

	var checkAndScreenScroll = func(counter int, x int) (int, int) {
		img := captureScreenShot()
		r, _, _, _ := imageInternal.GetPixelColor(img, 297, 320)
		fmt.Printf("r: %v\n", r)
		if r < 50 {
			scripts.ScrollDown(port, c, x)
		}
		return counter + 1, r
	}

	var checkAndClickScreenScroll = func(counter int) (int, int) {
		img := captureScreenShot()
		r, _, _, _ := imageInternal.GetPixelColor(img, 297, 342)
		if r < 50 {
			scripts.FastClick(port, c)
		}
		return counter + 1, r
	}

	countFilesInDir := func(dir string) (int, error) {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return 0, fmt.Errorf("не удалось прочитать папку: %v", err)
		}

		// Возвращаем количество файлов в папке
		return len(files), nil
	}

	// Функция для выполнения полного цикла скриншотов и OCR
	var performScreenshotAndOCR = func(buttonPressed bool) error {
		counter := 0
		maxCounter := 20
		scrollRPx := 26

		// Список для хранения всех скриншотов
		var screenshots []image.Image
		var smallScreenshots []image.Image

		img := captureScreenShot()
		screenshots = append(screenshots, img)

		scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
		fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)

		if scrollRPx > 26 {
			scrollRPx = 26
			for counter < maxCounter && scrollRPx < 50 {
				counter, scrollRPx = checkAndScreenScroll(counter, 1)
				if scrollRPx < 50 {
					img = captureScreenShot()
					screenshots = append(screenshots, img)
				}
			}

			scrollRPx = 26
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Scroll.X, Y: marginY + c.Click.Scroll.Y})
			for counter < maxCounter && scrollRPx < 50 {
				counter, scrollRPx = checkAndClickScreenScroll(counter)
				if scrollRPx < 50 {
					img = captureScreenShot()
					smallScreenshots = append(smallScreenshots, img)
				}
			}

			finalImage, _ := imageInternal.CombineImages(screenshots, smallScreenshots)
			combinedImg := imageInternal.CropOpacityPixel(finalImage)

			// Кадрирование комбинированного изображения
			bounds := combinedImg.Bounds()
			topCrop := 22 // По умолчанию обрезаем 22 пикселя сверху
			if buttonPressed {
				topCrop = 45 // Если кнопка была нажата, обрезаем 45 пикселей сверху
			}
			cropRect := image.Rect(40, topCrop, bounds.Dx()-17, bounds.Dy())
			croppedCombinedImg := combinedImg.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(cropRect)

			fileCount, _ := countFilesInDir("./imgs")
			fileName := fmt.Sprintf("%s/screenshot_combined_%d.png", "./imgs", fileCount)
			err := imageInternal.SaveCombinedImage(croppedCombinedImg, fileName)
			if err != nil {
				return err
			}

			scripts.ScrollUp(port, c, counter+5)

			result, err := ocr.RunOCR(fileName)
			if err != nil {
				fmt.Printf("Ошибка при выполнении OCR: %v\n", err)
				return err
			}

			fmt.Println(result)

			// Парсим результат OCR
			debugInfo, jsonData, rawText := ocr.ParseOCRResult(result)

			// Конвертируем изображение в байты
			imageBytes, err := imageToBytes(croppedCombinedImg)
			if err != nil {
				log.Printf("Ошибка конвертации изображения: %v", err)
				return err
			}

			// Сохраняем результат в базу данных
			_, err = saveOCRResultToDB(db, fileName, result, debugInfo, jsonData, rawText, imageBytes, c)
			if err != nil {
				log.Printf("Ошибка сохранения в БД: %v", err)
				return err
			}

			return nil
		}
		return fmt.Errorf("scrollRPx не превышает 26")
	}

	// Функция для проверки и клика по кнопке
	var checkAndClickButton = func(buttonX, buttonY int, buttonName string) bool {
		img := captureScreenShot()
		buttonRPx, _, _, _ := imageInternal.GetPixelColor(img, buttonX, buttonY)
		fmt.Printf("%s RPx: %v\n", buttonName, buttonRPx)
		return buttonRPx > 26
	}

	var captureScreenShotsWithScroll = func() bool {
		// Выполняем основной цикл скриншотов и OCR (без нажатия кнопок)
		if checkAndClickButton(c.Click.Button2.X, c.Click.Button2.Y, "listButton2") {
			err := performScreenshotAndOCR(true)
			if err != nil {
				return false
			}
		} else {
			err := performScreenshotAndOCR(false)
			if err != nil {
				return false
			}
		}

		// Проверяем и кликаем по кнопке 2
		if checkAndClickButton(c.Click.Button2.X, c.Click.Button2.Y, "listButton2") {
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button2.X, Y: marginY + c.Click.Button2.Y})

			// Повторяем цикл для кнопки 2 (с нажатием кнопки)
			err = performScreenshotAndOCR(true)
			if err != nil {
				return false
			}
		}

		// Проверяем и кликаем по кнопке 3
		if checkAndClickButton(c.Click.Button3.X, c.Click.Button3.Y, "listButton3") {
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button3.X, Y: marginY + c.Click.Button3.Y})

			// Повторяем цикл для кнопки 3 (с нажатием кнопки)
			err = performScreenshotAndOCR(true)
			if err != nil {
				return false
			}
		}

		// Проверяем и кликаем по кнопке 4
		if checkAndClickButton(c.Click.Button4.X, c.Click.Button4.Y, "listButton4") {
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button4.X, Y: marginY + c.Click.Button4.Y})

			// Повторяем цикл для кнопки 4 (с нажатием кнопки)
			err = performScreenshotAndOCR(true)
			if err != nil {
				return false
			}
		}

		return true
	}

	var clickItem = func(item config.Coordinates) {
		scripts.ClickCoordinates(port, c, item)
		combinedSaved := captureScreenShotsWithScroll()
		if !combinedSaved {
			saveScreenShot()
		}

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
	}

	var clickEveryItemAnsScreenShot = func(img image.Image) {
		// прокликиваем первую страницу
		// points := imageInternal.FindItemPositionsByTextColor(img, 80)
		// if len(points) > 2 {
		// 	for _, point := range points {
		// 		clickItem(config.Coordinates{Y: point.Y + marginY, X: marginX + point.X})
		// 	}
		// }

		clickItem(config.Coordinates{X: marginX + c.Click.Item3.X, Y: marginY + c.Click.Item3.Y})
	}

	// берем в фокус и делаем скрин
	scripts.ClickCoordinates(port, c, c.Click.Item1)
	img = captureScreenShot()
	clickEveryItemAnsScreenShot(img)

	// // берем в фокус
	// scripts.ClickCoordinates(port, c, c.Click.Item1)

	// cycles := 0
	// for cycles < 20 {
	// 	img := captureScreenShot()
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button2.X, Y: marginY + c.Click.Button2.Y})
	// 	img = captureScreenShot()
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button3.X, Y: marginY + c.Click.Button3.Y})
	// 	img = captureScreenShot()
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button4.X, Y: marginY + c.Click.Button4.Y})
	// 	img = captureScreenShot()
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button5.X, Y: marginY + c.Click.Button5.Y})
	// 	img = captureScreenShot()
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
	// 	img = captureScreenShot()
	// 	clickEveryItemAnsScreenShot(img)

	// 	img = captureScreenShot()
	// 	SixButtonPx, _, _, _ := imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
	// 	maxSixButtonClicks := 0

	// 	for SixButtonPx > 30 && maxSixButtonClicks < 50 {
	// 		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
	// 		img = captureScreenShot()
	// 		clickEveryItemAnsScreenShot(img)
	// 		img = captureScreenShot()
	// 		SixButtonPx, _, _, _ = imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
	// 		maxSixButtonClicks += 1
	// 	}

	// 	scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
	// 	// scripts.ClickCoordinates(port, c, config.Coordinates{X: 35, Y: 107})

	// 	cycles += 1
	// }

}

// saveOCRResultToDB сохраняет результат OCR в базу данных
func saveOCRResultToDB(db *sql.DB, imagePath, ocrResult string, debugInfo, jsonData string, rawText string, imageData []byte, cfg *config.Config) (int, error) {
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
