package cycle_listed_items

import (
	"fmt"
	"image"
	"shnyr/internal/click_manager"
	"shnyr/internal/config"
	"shnyr/internal/database"
	"shnyr/internal/interrupt"
	"shnyr/internal/logger"
	"shnyr/internal/ocr"
	"shnyr/internal/screenshot"
)

// processItemListPage обрабатывает отдельный предмет со всеми его кнопками
func processItemListPage(c *config.Config, screenshotManager *screenshot.ScreenshotManager, ocrManager *ocr.OCRManager, dbManager *database.DatabaseManager, clickManager *click_manager.ClickManager, loggerManager *logger.LoggerManager, interruptManager *interrupt.InterruptManager, isFirstCycle bool, currentItem string, itemCategory string) error {
	loggerManager.Info("🎯 processItemListPage: предмет='%s', категория='%s', первый_цикл=%v", currentItem, itemCategory, isFirstCycle)

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

	// Определяем тип предмета на основе категории
	isConsumable := itemCategory == "buy_consumables" || itemCategory == "sell_consumables"
	isEquipment := itemCategory == "buy_equipment" || itemCategory == "sell_equipment"
	isBuy := itemCategory == "buy_consumables" || itemCategory == "buy_equipment"
	isSell := itemCategory == "sell_consumables" || itemCategory == "sell_equipment"

	if isConsumable {
		loggerManager.Info("🍶 Предмет '%s' является расходником (%s), пропускаем кнопки страниц", currentItem, itemCategory)
	} else if isEquipment {
		loggerManager.Info("⚔️ Предмет '%s' является экипировкой (%s), обрабатываем кнопки страниц", currentItem, itemCategory)
	} else {
		loggerManager.Info("❓ Предмет '%s' имеет неизвестную категорию (%s), обрабатываем как экипировку", currentItem, itemCategory)
		isEquipment = true
	}

	if isBuy {
		loggerManager.Info("💰 Предмет '%s' предназначен для скупки", currentItem)
	} else if isSell {
		loggerManager.Info("💸 Предмет '%s' предназначен для продажи", currentItem)
	}

	// Обрабатываем каждый найденный предмет начиная с указанного
	for i := startIndex; i < len(itemCoordinates); i++ {
		coordinate := itemCoordinates[i]

		// Проверяем сигнал прерывания в начале обработки каждого предмета
		interrupted, _, checkErr := checkInterruption(interruptManager, dbManager, loggerManager)
		if checkErr != nil {
			loggerManager.Info("⏹️ Прерывание по запросу пользователя")
			return fmt.Errorf("прерывание по запросу пользователя")
		}
		if interrupted {
			loggerManager.Info("⏹️ Прерывание по запросу пользователя")
			return fmt.Errorf("прерывание по запросу пользователя")
		}

		loggerManager.Info("📍 Обрабатываем предмет %d/%d в координатах: %v", i+1, len(itemCoordinates), coordinate)

		// кликаем по предмету
		clickManager.ClickCoordinates(coordinate)

		// получаем статус страницы
		pageStatus := screenshotManager.GetPageStatus(c)

		// обрабатываем первую страницу предмета
		err := processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager, currentItem, itemCategory)
		if err != nil {
			loggerManager.LogError(err, "Ошибка при обработке первой страницы")
			return err
		}

		// Для consumables пропускаем обработку кнопок страниц
		if !isConsumable {
			if pageStatus.Buttons.Button2Active {
				// кликаем по кнопке 2
				clickManager.ClickCoordinates(image.Point{X: c.Click.Button2.X, Y: c.Click.Button2.Y})

				// обрабатываем страницу кнопки 2
				err = processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager, currentItem, itemCategory)
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
				err = processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager, currentItem, itemCategory)
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
				err = processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager, currentItem, itemCategory)
				if err != nil {
					loggerManager.LogError(err, "Ошибка при обработке кнопки 4")
					return err
				}
			}

			// обновляем статус страницы тк он мог устареть
			pageStatus = screenshotManager.GetPageStatus(c)
			if pageStatus.Buttons.Button5Active {
				// кликаем по кнопке 5
				clickManager.ClickCoordinates(image.Point{X: c.Click.Button5.X, Y: c.Click.Button5.Y})

				// обрабатываем страницу кнопки 5
				err = processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager, currentItem, itemCategory)
				if err != nil {
					loggerManager.LogError(err, "Ошибка при обработке кнопки 5")
					return err
				}
			}

			// обновляем статус страницы тк он мог устареть
			pageStatus = screenshotManager.GetPageStatus(c)
			if pageStatus.Buttons.Button6Active {
				// кликаем по кнопке 6
				clickManager.ClickCoordinates(image.Point{X: c.Click.Button6.X, Y: c.Click.Button6.Y})

				// обрабатываем страницу кнопки 6
				err = processItemPageWithButtonLogic(c, screenshotManager, ocrManager, dbManager, loggerManager, currentItem, itemCategory)
				if err != nil {
					loggerManager.LogError(err, "Ошибка при обработке кнопки 6")
					return err
				}
			}
		} else {
			loggerManager.Info("🍶 Пропускаем обработку кнопок страниц для расходника")
		}

		// кликаем по back
		clickManager.ClickCoordinates(image.Point{X: c.Click.Back.X, Y: c.Click.Back.Y})
	}
	return nil
}

// processItemsByCategory обрабатывает предметы определенной категории (buy или sell)
func processItemsByCategory(c *config.Config, screenshotManager *screenshot.ScreenshotManager, ocrManager *ocr.OCRManager, dbManager *database.DatabaseManager, clickManager *click_manager.ClickManager, loggerManager *logger.LoggerManager, interruptManager *interrupt.InterruptManager, category string, cycles int, startButtonIndex int) error {
	// Получаем список предметов определенной категории
	itemList, err := dbManager.GetItemsByCategory(category)
	if err != nil {
		loggerManager.LogError(err, fmt.Sprintf("Ошибка получения списка предметов категории %s", category))
		return err
	}

	loggerManager.Info("🔍 DEBUG: GetItemsByCategory('%s') вернул %d предметов: %v", category, len(itemList), itemList)

	if len(itemList) == 0 {
		loggerManager.Info("📋 Нет предметов для категории %s", category)
		return nil
	}

	loggerManager.Info("📋 Обрабатываем %d предметов категории %s", len(itemList), category)

	for i, item := range itemList {
		interrupted, _, checkErr := checkInterruption(interruptManager, dbManager, loggerManager)
		if checkErr != nil {
			loggerManager.Info("⏹️ Прерывание по запросу пользователя")
			return fmt.Errorf("прерывание по запросу пользователя")
		}
		if interrupted {
			loggerManager.Info("⏹️ Прерывание по запросу пользователя")
			return fmt.Errorf("прерывание по запросу пользователя")
		}

		loggerManager.Info("🔍 Обрабатываем предмет %d/%d: %s (категория: %s)", i+1, len(itemList), item, category)

		// Копируем название предмета в буфер обмена
		clickManager.CopyToClipboard(item)

		// Вставляем название предмета
		clickManager.Paste()

		// кликаем на поиск
		clickManager.ClickCoordinates(image.Point{X: 120, Y: 240})

		// обрабатываем все активные кнопки в цикле
		var buttonIndex int
		if cycles == 0 {
			// В первом цикле начинаем с указанной начальной кнопки
			buttonIndex = startButtonIndex
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
			err := processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, cycles == 0 && buttonIndex == startButtonIndex, item, category)
			if err != nil {
				if err.Error() == "прерывание по запросу пользователя" {
					loggerManager.Info("⏹️ Завершение работы по прерыванию")
					return err
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
				if buttonIndex == startButtonIndex && cycles == 0 {
					loggerManager.Info("🔘 Обрабатываем начальную кнопку %d (первый цикл)", buttonIndex)
					clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
					err := processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, true, item, category)
					if err != nil {
						if err.Error() == "прерывание по запросу пользователя" {
							loggerManager.Info("⏹️ Завершение работы по прерыванию")
							return err
						}
						loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
					}

					// Для кнопки 6 продолжаем нажимать пока она активна
					if buttonIndex == 6 {
						for screenshotManager.CheckButtonActiveByPixel(buttonX, 35) {
							loggerManager.Info("🔘 Повторно обрабатываем кнопку 6")
							clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
							err = processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false, item, category)
							if err != nil {
								if err.Error() == "прерывание по запросу пользователя" {
									loggerManager.Info("⏹️ Завершение работы по прерыванию")
									return err
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
					err := processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false, item, category)
					if err != nil {
						if err.Error() == "прерывание по запросу пользователя" {
							loggerManager.Info("⏹️ Завершение работы по прерыванию")
							return err
						}
						loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
					}

					// Для кнопки 6 продолжаем нажимать пока она активна
					if buttonIndex == 6 {
						for screenshotManager.CheckButtonActiveByPixel(buttonX, 35) {
							loggerManager.Info("🔘 Повторно обрабатываем кнопку 6")
							clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
							err = processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false, item, category)
							if err != nil {
								if err.Error() == "прерывание по запросу пользователя" {
									loggerManager.Info("⏹️ Завершение работы по прерыванию")
									return err
								}
								loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
							}
						}
						loggerManager.Info("🔍 Кнопка 6 больше неактивна, завершаем обработку")
						break
					}
				} else {
					// Для всех остальных кнопок проверяем активность
					loggerManager.Info("🔍 Проверяем кнопку %d (начальная кнопка: %d, цикл: %d)", buttonIndex, startButtonIndex, cycles+1)

					if screenshotManager.CheckButtonActiveByPixel(buttonX, 35) {
						loggerManager.Info("🔘 Обрабатываем кнопку %d", buttonIndex)
						clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
						err := processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false, item, category)
						if err != nil {
							if err.Error() == "прерывание по запросу пользователя" {
								loggerManager.Info("⏹️ Завершение работы по прерыванию")
								return err
							}
							loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
						}

						// Для кнопки 6 продолжаем нажимать пока она активна
						if buttonIndex == 6 {
							for screenshotManager.CheckButtonActiveByPixel(buttonX, 35) {
								loggerManager.Info("🔘 Повторно обрабатываем кнопку 6")
								clickManager.ClickCoordinates(image.Point{X: buttonX, Y: buttonY})
								err = processItemListPage(c, screenshotManager, ocrManager, dbManager, clickManager, loggerManager, interruptManager, false, item, category)
								if err != nil {
									if err.Error() == "прерывание по запросу пользователя" {
										loggerManager.Info("⏹️ Завершение работы по прерыванию")
										return err
									}
									loggerManager.LogError(err, "Ошибка при обработке страницы с предметами")
								}
							}
							loggerManager.Info("🔍 Кнопка 6 больше неактивна, завершаем обработку")
							break
						}
					} else {
						loggerManager.Info("🔍 Кнопка %d неактивна, пропускаем", buttonIndex)
					}
				}
				buttonIndex++
			}
		}

		clickManager.ClickCoordinates(image.Point{X: c.Click.Back.X, Y: c.Click.Back.Y})
	}

	return nil
}
