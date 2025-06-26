package cycle_listed_items

import (
	"bytes"
	"image"
	"image/png"
	"shnyr/internal/config"
	"shnyr/internal/database"
	"shnyr/internal/logger"
	"shnyr/internal/ocr"
	"shnyr/internal/screenshot"
)

var processItemPage = func(c *config.Config,
	pageStatus screenshot.PageStatus,
	screenshotManager *screenshot.ScreenshotManager,
	loggerManager *logger.LoggerManager) (image.Image, string, error) {

	var finalImg image.Image
	// сохраняем окно покупки в переменную
	img, err := screenshotManager.CaptureScreenShot()
	if err != nil {
		loggerManager.LogError(err, "Ошибка при захвате скриншота")
		return nil, "", err
	}

	// выводим общий лог статуса страницы
	loggerManager.Info("📄 Статус страницы: кнопка1=%v, кнопка2=%v, кнопка3=%v, кнопка4=%v, кнопка5=%v, кнопка6=%v, скролл=%v",
		pageStatus.Buttons.Button1Active,
		pageStatus.Buttons.Button2Active,
		pageStatus.Buttons.Button3Active,
		pageStatus.Buttons.Button4Active,
		pageStatus.Buttons.Button5Active,
		pageStatus.Buttons.Button6Active,
		pageStatus.HasScroll)

	// если скролла нет, сохраняем как финальное изображение
	if !pageStatus.HasScroll {
		finalImg = img
	}

	// если скролл есть, собираем изображение по кусочкам
	if pageStatus.HasScroll {
		// собираем изображение по кусочкам
		img, err := screenshotManager.PerformScreenshotWithScroll(pageStatus, c)
		if err != nil {
			loggerManager.LogError(err, "Ошибка в цикле скриншотов со скроллом")
			return nil, "", err
		}
		finalImg = img
	}

	buttonExist := pageStatus.Buttons.Button1Active || pageStatus.Buttons.Button2Active || pageStatus.Buttons.Button3Active || pageStatus.Buttons.Button4Active || pageStatus.Buttons.Button5Active || pageStatus.Buttons.Button6Active

	// обрезаем изображение с помощью ScreenshotManager
	croppedFinalImg := screenshotManager.CropImageForText(finalImg, c, buttonExist)

	// сохраняем croppedFinalImg
	savedImagePath, err := screenshotManager.SaveImage(croppedFinalImg, "sreenshot.png", c.SaveAllScreenshots, loggerManager)

	if err != nil {
		loggerManager.LogError(err, "Ошибка сохранения изображения")
	} else {
		loggerManager.Info("🖼️ Изображение сохранено: %s", savedImagePath)
	}

	return croppedFinalImg, savedImagePath, nil
}

// processItemPageWithButtonLogic обрабатывает страницу с кнопкой (обработка изображения, OCR, сохранение в БД)
func processItemPageWithButtonLogic(c *config.Config, screenshotManager *screenshot.ScreenshotManager, ocrManager *ocr.OCRManager, dbManager *database.DatabaseManager, loggerManager *logger.LoggerManager, currentItem string, itemCategory string) error {
	// получаем статус страницы
	pageStatus := screenshotManager.GetPageStatus(c)

	// сохраняем изображение страницы предмета
	croppedFinalImg, savedImgPath, err := processItemPage(c, pageStatus, screenshotManager, loggerManager)
	if err != nil {
		loggerManager.LogError(err, "Ошибка при обработке страницы")
		return err
	}

	// проводим OCR картинки
	result, debugInfo, jsonData, rawText, err := ocrManager.ProcessImage(savedImgPath)
	if err != nil {
		loggerManager.LogError(err, "Ошибка при проведении OCR")
		return err
	}

	// сохраняем результат в базу
	var imgBytes bytes.Buffer
	png.Encode(&imgBytes, croppedFinalImg)

	loggerManager.Info("💾 Сохраняем OCR результат для предмета '%s' с категорией '%s'", currentItem, itemCategory)

	num, err := dbManager.SaveOCRResultToDB(savedImgPath, result, debugInfo, jsonData, rawText, imgBytes.Bytes(), c, itemCategory, currentItem)
	if err != nil {
		loggerManager.LogError(err, "Ошибка при сохранении результата в базу")
		return err
	}
	loggerManager.Info("🔍 OCR результат сохранен с ID: %d", num)

	return nil
}
