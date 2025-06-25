package cycle_all_items

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"shnyr/internal/click_manager"
	"shnyr/internal/config"
	"shnyr/internal/database"
	"shnyr/internal/interrupt"
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
	loggerManager.Info("📄 Статус страницы: кнопка1=%v, кнопка2=%v, кнопка3=%v, кнопка4=%v, скролл=%v",
		pageStatus.Buttons.Button1Active,
		pageStatus.Buttons.Button2Active,
		pageStatus.Buttons.Button3Active,
		pageStatus.Buttons.Button4Active,
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

	buttonExist := pageStatus.Buttons.Button1Active || pageStatus.Buttons.Button2Active || pageStatus.Buttons.Button3Active || pageStatus.Buttons.Button4Active
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

// processButtonPage обрабатывает страницу с кнопкой (обработка изображения, OCR, сохранение в БД)
func processItemPageWithButtonLogic(c *config.Config, screenshotManager *screenshot.ScreenshotManager, ocrManager *ocr.OCRManager, dbManager *database.DatabaseManager, loggerManager *logger.LoggerManager) error {
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
	num, err := dbManager.SaveOCRResultToDB(savedImgPath, result, debugInfo, jsonData, rawText, imgBytes.Bytes(), c)
	if err != nil {
		loggerManager.LogError(err, "Ошибка при сохранении результата в базу")
		return err
	}
	loggerManager.Info("🔍 OCR результат сохранен с ID: %d", num)

	return nil
}

// processItem обрабатывает отдельный предмет со всеми его кнопками
func processItemListPage(c *config.Config, screenshotManager *screenshot.ScreenshotManager, ocrManager *ocr.OCRManager, dbManager *database.DatabaseManager, clickManager *click_manager.ClickManager, loggerManager *logger.LoggerManager, interruptManager *interrupt.InterruptManager, isFirstCycle bool) error {
	itemCoordinates, err := screenshotManager.GetItemListItemsCoordinates()
	if err != nil {
		loggerManager.LogError(err, "Ошибка при поиске координат первой страницы")
		return err
	}

	// Определяем стартовый индекс предмета только для первого цикла
	var startIndex int
	if isFirstCycle {
		// Для первого цикла используем указанный стартовый предмет
		startIndex = c.StartItemIndex - 1 // Номер предмета начинается с 1, индекс с 0
		if startIndex < 0 {
			loggerManager.Info("⚠️ Номер предмета %d некорректен, начинаем с первого", c.StartItemIndex)
			startIndex = 0
		}
		if startIndex >= len(itemCoordinates) {
			loggerManager.Info("⚠️ Номер предмета %d превышает количество предметов (%d), начинаем с первого", c.StartItemIndex, len(itemCoordinates))
			startIndex = 0
		}
		loggerManager.Info("📍 Первый цикл: начинаем с предмета %d из %d", startIndex+1, len(itemCoordinates))
	} else {
		// Для последующих циклов начинаем с первого предмета
		startIndex = 0
		loggerManager.Info("📍 Последующий цикл: начинаем с первого предмета из %d", len(itemCoordinates))
	}

	// Обрабатываем каждый найденный предмет начиная с указанного
	for i := startIndex; i < len(itemCoordinates); i++ {
		coordinate := itemCoordinates[i]

		// Проверяем сигнал прерывания в начале обработки каждого предмета
		select {
		case <-interruptManager.GetScriptInterruptChan():
			loggerManager.Info("⏹️ Прерывание script1 по запросу пользователя")
			return fmt.Errorf("прерывание по запросу пользователя")
		default:
		}

		loggerManager.Info("📍 Обрабатываем предмет %d/%d в координатах: %v", i+1, len(itemCoordinates), coordinate)

		// кликаем по предмету
		clickManager.ClickCoordinates(coordinate)

		// получаем статус страницы
		pageStatus := screenshotManager.GetPageStatus(c)

		// обрабатываем первую страницу предмета
		err := processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager)
		if err != nil {
			loggerManager.LogError(err, "Ошибка при обработке первой страницы")
			return err
		}

		if pageStatus.Buttons.Button2Active {
			// кликаем по кнопке 2
			clickManager.ClickCoordinates(image.Point{X: c.Click.Button2.X, Y: c.Click.Button2.Y})

			// обрабатываем страницу кнопки 2
			err = processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager)
			if err != nil {
				loggerManager.LogError(err, "Ошибка при обработке кнопки 2")
				return err
			}
		}

		// обновляем статус страницы тк он мог устареть
		pageStatus = screenshotManager.GetPageStatus(c)
		if pageStatus.Buttons.Button3Active {
			// кликаем по кнопке 3
			clickManager.ClickCoordinates(image.Point{X: c.Click.Button3.X, Y: c.Click.Button3.Y})

			// обрабатываем страницу кнопки 3
			err = processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager)
			if err != nil {
				loggerManager.LogError(err, "Ошибка при обработке кнопки 3")
				return err
			}
		}

		// обновляем статус страницы тк он мог устареть
		pageStatus = screenshotManager.GetPageStatus(c)
		if pageStatus.Buttons.Button4Active {
			// кликаем по кнопке 4
			clickManager.ClickCoordinates(image.Point{X: c.Click.Button4.X, Y: c.Click.Button4.Y})

			// обрабатываем страницу кнопки 4
			err = processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager)
			if err != nil {
				loggerManager.LogError(err, "Ошибка при обработке кнопки 4")
				return err
			}
		}

		// кликаем по back
		clickManager.ClickCoordinates(image.Point{X: c.Click.Back.X, Y: c.Click.Back.Y})
	}
	return nil
}

var Run = func(c *config.Config, screenshotManager *screenshot.ScreenshotManager, dbManager *database.DatabaseManager, ocrManager *ocr.OCRManager, clickManager *click_manager.ClickManager, loggerManager *logger.LoggerManager, interruptManager *interrupt.InterruptManager) {
	// берем окно L2 в фокус
	clickManager.FocusL2Window()

	// цикл обработки страниц с предметами, количество полных проходов хранится в конфиге в переменной max_cycles_items_list
	for cycles := 0; cycles < c.MaxCyclesItemsList; cycles++ {
		loggerManager.Info("🔄 Проход %d из %d", cycles+1, c.MaxCyclesItemsList)

		select {
		case <-interruptManager.GetScriptInterruptChan():
			loggerManager.Info("⏹️ Прерывание script1 по запросу пользователя")
			return
		default:
		}

		// обрабатываем все активные кнопки в цикле
		var buttonIndex int
		if cycles == 0 {
			// В первом цикле начинаем с указанной начальной кнопки
			buttonIndex = c.StartButtonIndex
		} else {
			// В последующих циклах начинаем с первой кнопки
			buttonIndex = 1
		}

		// Проверяем есть ли вообще активные кнопки
		hasActiveButtons := false
		for checkButton := 1; checkButton <= 6; checkButton++ {
			var checkButtonX int
			switch checkButton {
			case 1:
				checkButtonX = c.Click.Button1.X
			case 2:
				checkButtonX = c.Click.Button2.X
			case 3:
				checkButtonX = c.Click.Button3.X
			case 4:
				checkButtonX = c.Click.Button4.X
			case 5:
				checkButtonX = c.Click.Button5.X
			case 6:
				checkButtonX = c.Click.Button6.X
			}

			if screenshotManager.CheckButtonActiveByPixel(checkButtonX, 35) {
				hasActiveButtons = true
				break
			}
		}

		if !hasActiveButtons {
			loggerManager.Info("🔍 Активных кнопок не найдено, обрабатываем список предметов без кнопок")
			err := processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, cycles == 0 && buttonIndex == c.StartButtonIndex)
			if err != nil {
				if err.Error() == "прерывание по запросу пользователя" {
					loggerManager.Info("⏹️ Завершение работы по прерыванию")
					return
				}
				loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
			}
		} else {
			// Обрабатываем кнопки как обычно
			for buttonIndex <= 6 {
				var buttonX, buttonY int
				switch buttonIndex {
				case 1:
					buttonX, buttonY = c.Click.Button1.X, c.Click.Button1.Y
				case 2:
					buttonX, buttonY = c.Click.Button2.X, c.Click.Button2.Y
				case 3:
					buttonX, buttonY = c.Click.Button3.X, c.Click.Button3.Y
				case 4:
					buttonX, buttonY = c.Click.Button4.X, c.Click.Button4.Y
				case 5:
					buttonX, buttonY = c.Click.Button5.X, c.Click.Button5.Y
				case 6:
					buttonX, buttonY = c.Click.Button6.X, c.Click.Button6.Y
				}

				// Для начальной кнопки в первом цикле не проверяем активность - сразу кликаем
				if buttonIndex == c.StartButtonIndex && cycles == 0 {
					loggerManager.Info("🔘 Обрабатываем начальную кнопку %d (первый цикл)", buttonIndex)
					clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
					err := processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, true)
					if err != nil {
						if err.Error() == "прерывание по запросу пользователя" {
							loggerManager.Info("⏹️ Завершение работы по прерыванию")
							return
						}
						loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
					}

					// Для кнопки 6 продолжаем нажимать пока она активна
					if buttonIndex == 6 {
						for screenshotManager.CheckButtonActiveByPixel(buttonX, 35) {
							loggerManager.Info("🔘 Повторно обрабатываем кнопку 6")
							clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
							err = processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false)
							if err != nil {
								if err.Error() == "прерывание по запросу пользователя" {
									loggerManager.Info("⏹️ Завершение работы по прерыванию")
									return
								}
								loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
							}
						}
						loggerManager.Info("🔍 Кнопка 6 больше неактивна, завершаем обработку")
						break
					}
				} else if buttonIndex == 1 {
					// Кнопка 1 всегда кликается без проверки активности
					loggerManager.Info("🔘 Обрабатываем кнопку 1 (всегда активна)")
					clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
					err := processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false)
					if err != nil {
						if err.Error() == "прерывание по запросу пользователя" {
							loggerManager.Info("⏹️ Завершение работы по прерыванию")
							return
						}
						loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
					}

					// Для кнопки 6 продолжаем нажимать пока она активна
					if buttonIndex == 6 {
						for screenshotManager.CheckButtonActiveByPixel(buttonX, 35) {
							loggerManager.Info("🔘 Повторно обрабатываем кнопку 6")
							clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
							err = processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false)
							if err != nil {
								if err.Error() == "прерывание по запросу пользователя" {
									loggerManager.Info("⏹️ Завершение работы по прерыванию")
									return
								}
								loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
							}
						}
						loggerManager.Info("🔍 Кнопка 6 больше неактивна, завершаем обработку")
						break
					}
				} else {
					// Для всех остальных кнопок проверяем активность
					loggerManager.Info("🔍 Проверяем кнопку %d (начальная кнопка: %d, цикл: %d)", buttonIndex, c.StartButtonIndex, cycles+1)

					if screenshotManager.CheckButtonActiveByPixel(buttonX, 35) {
						loggerManager.Info("🔘 Обрабатываем кнопку %d", buttonIndex)
						clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
						err := processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false)
						if err != nil {
							if err.Error() == "прерывание по запросу пользователя" {
								loggerManager.Info("⏹️ Завершение работы по прерыванию")
								return
							}
							loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
						}

						// Для кнопки 6 продолжаем нажимать пока она активна
						if buttonIndex == 6 {
							for screenshotManager.CheckButtonActiveByPixel(buttonX, 35) {
								loggerManager.Info("🔘 Повторно обрабатываем кнопку 6")
								clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
								err = processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false)
								if err != nil {
									if err.Error() == "прерывание по запросу пользователя" {
										loggerManager.Info("⏹️ Завершение работы по прерыванию")
										return
									}
									loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
								}
							}
							loggerManager.Info("🔍 Кнопка 6 больше неактивна, завершаем обработку")
							break
						}
					} else {
						loggerManager.Info("🔍 Кнопка %d неактивна, завершаем обработку", buttonIndex)
						break
					}
				}
				buttonIndex++
			}
		}

		loggerManager.Info("✅ Обработали все элементы на странице %d из %d", cycles+1, c.MaxCyclesItemsList)

		// кликает по back
		clickManager.ClickCoordinates(image.Point{X: c.Click.Back.X, Y: c.Click.Back.Y})

		// возвращается в список
		clickManager.ClickCoordinates(image.Point{X: c.Click.Button1.X, Y: c.Click.Button1.Y})
	}
}
