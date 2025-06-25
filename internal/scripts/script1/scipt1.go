package scpript1

import (
	"fmt"
	"image"
	"log"
	"octopus/internal/click_manager"
	"octopus/internal/config"
	"octopus/internal/database"
	imageInternal "octopus/internal/image"
	"octopus/internal/ocr"
	"octopus/internal/screenshot"
)

// clickPageButton кликает по кнопке
func clickPageButton(c *config.Config, clickManager *click_manager.ClickManager, dbManager *database.DatabaseManager, buttonName string, buttonCoords config.Coordinates, isActive bool, marginX, marginY int) {
	if isActive {
		log.Printf("🔘 Кликаем по %s...", buttonName)
		clickManager.ClickCoordinates(config.Coordinates{X: marginX + buttonCoords.X, Y: marginY + buttonCoords.Y})
	} else {
		log.Printf("⏭️ %s неактивен, пропускаем", buttonName)
	}
}

// getScreenshotOfItemPage делает скриншот страницы предмета
func getScreenshotOfItemPage(c *config.Config, clickManager *click_manager.ClickManager, screenshotManager *screenshot.ScreenshotManager, buttonStatus imageInternal.ButtonStatus, scrollRPx int, marginX, marginY int) (image.Image, error) {
	// Если нет скролла, делаем обычный скриншот
	if scrollRPx <= 26 {
		log.Println("❌ Скролл не найден (scrollRPx <= 26), делаем обычный скриншот")
		return screenshotManager.CaptureScreenShot(), nil
	}

	// Выполняем основной цикл скриншотов и OCR (без нажатия кнопок)
	fmt.Println("🔄 Выполняем цикл скриншотов со скроллом...")
	if buttonStatus.Button2Active {
		img, _, err := clickManager.PerformScreenshotWithScroll(true)
		if err != nil {
			log.Printf("❌ Ошибка в основном цикле скриншотов со скроллом: %v\n", err)
			return img, err
		}
		return img, nil
	} else {
		img, _, err := clickManager.PerformScreenshotWithScroll(false)
		if err != nil {
			log.Printf("❌ Ошибка в цикле скриншотов со скроллом: %v\n", err)
			return img, err
		}
		return img, nil
	}
}

// processItemPages обрабатывает отдельный предмет (клик, проверка скролла, обработка скриншотов)
func processItemPages(c *config.Config, clickManager *click_manager.ClickManager, screenshotManager *screenshot.ScreenshotManager, dbManager *database.DatabaseManager, ocrManager *ocr.OCRManager, point image.Point, marginX, marginY int) {

	img := screenshotManager.CaptureScreenShot()

	// Сначала проверяем, есть ли скролл вообще
	scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
	log.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)

	// Проверяем наличие всех кнопок
	log.Println("🔍 Проверяем наличие кнопок...")
	buttonStatus := imageInternal.CheckAllButtonsStatus(img, c, marginX, marginY)

	// Обрабатываем страницу предмета
	itemPageImg, err := getScreenshotOfItemPage(c, clickManager, screenshotManager, buttonStatus, scrollRPx, marginX, marginY)
	if err != nil {
		log.Printf("❌ Ошибка получения скриншота: %v", err)
		return
	}

	result, debugInfo, jsonData, rawText, err := ocrManager.ProcessImage(itemPageImg, "itemPageImg")
	if err != nil {
		log.Printf("❌ Ошибка OCR: %v", err)
	}

	// Сохраняем результат в базу данных
	imageBytes, err := imageInternal.ImageToBytes(itemPageImg)
	if err != nil {
		log.Printf("❌ Ошибка конвертации изображения: %v", err)
		return
	}

	_, err = dbManager.SaveOCRResultToDB("itemPageImg", result, debugInfo, jsonData, rawText, imageBytes, c)
	if err != nil {
		log.Printf("Ошибка сохранения в БД: %v", err)
	}

	// Кликаем Back только после последней существующей кнопки
	log.Println("🔙 Кликаем по кнопке Back...")
	clickManager.ClickCoordinates(config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
	log.Println("✅ Back клик выполнен")
}

var Run = func(c *config.Config, screenshotManager *screenshot.ScreenshotManager, dbManager *database.DatabaseManager, ocrManager *ocr.OCRManager, clickManager *click_manager.ClickManager, marginX, marginY int) {
	// берем окно в фокус (кликаем по любым координатам)
	clickManager.FocusL2Window()

	// цикл обработки страниц
	cycles := 0

	for cycles < c.MaxCyclesItemsList {
		// обрабатываем первую страницу
		itemCoordinates, err := screenshotManager.GetItemListItemsCoordinates()
		if err != nil {
			log.Printf("Ошибка при поиске координат первой страницы: %v", err)
		}

		// Обрабатываем каждый найденный элемент
		for _, coordinate := range itemCoordinates {
			log.Printf("📍 Обрабатываем элемент в координатах: %v", coordinate)
			processItemPages(c, clickManager, screenshotManager, dbManager, ocrManager, coordinate, marginX, marginY)
		}

		cycles += 1
	}
}
