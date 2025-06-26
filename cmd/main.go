package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"shnyr/internal/arduino"
	"shnyr/internal/click_manager"
	"shnyr/internal/config"
	"shnyr/internal/database"
	imageInternal "shnyr/internal/image"
	"shnyr/internal/interrupt"
	"shnyr/internal/logger"
	"shnyr/internal/ocr"
	"shnyr/internal/screenshot"
	cycleAllItems "shnyr/internal/scripts/cycle_all_items"
	cycleListedItems "shnyr/internal/scripts/cycle_listed_items"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tarm/serial"
)

func getStartButtonFromConsole() int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("🔘 Введите номер стартовой кнопки (1-6): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Ошибка чтения ввода:", err)
			continue
		}

		// Убираем пробелы и переносы строк
		input = strings.TrimSpace(input)

		// Проверяем на пустой ввод
		if input == "" {
			fmt.Println("⚠️ Введите число от 1 до 6")
			continue
		}

		// Парсим число
		buttonNum, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("❌ Введите корректное число")
			continue
		}

		// Проверяем диапазон
		if buttonNum < 1 || buttonNum > 6 {
			fmt.Println("❌ Номер кнопки должен быть от 1 до 6")
			continue
		}

		return buttonNum
	}
}

func getStartItemFromConsole() int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("📍 Введите номер стартового предмета (1 для начала с первого): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Ошибка чтения ввода:", err)
			continue
		}

		// Убираем пробелы и переносы строк
		input = strings.TrimSpace(input)

		// Проверяем на пустой ввод
		if input == "" {
			fmt.Println("⚠️ Введите число от 1 и больше")
			continue
		}

		// Парсим число
		itemNum, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("❌ Введите корректное число")
			continue
		}

		// Проверяем диапазон
		if itemNum < 1 {
			fmt.Println("❌ Номер предмета должен быть 1 или больше")
			continue
		}

		return itemNum
	}
}

// updateStatus обновляет статус в базе данных
func updateStatus(db *sql.DB, status string) error {
	_, err := db.Exec("INSERT INTO status (current_status) VALUES (?)", status)
	return err
}

// addAction добавляет действие в базу данных
func addAction(db *sql.DB, action string) error {
	_, err := db.Exec("INSERT INTO actions (action) VALUES (?)", action)
	return err
}

// updateLatestPendingAction помечает последнее невыполненное действие как выполненное
func updateLatestPendingAction(db *sql.DB) error {
	_, err := db.Exec("UPDATE actions SET executed = TRUE WHERE id = (SELECT id FROM actions WHERE executed = FALSE ORDER BY created_at DESC LIMIT 1)")
	return err
}

// getLatestPendingAction возвращает последнее невыполненное действие
func getLatestPendingAction(db *sql.DB) (string, error) {
	var action string
	err := db.QueryRow("SELECT action FROM actions WHERE executed = FALSE ORDER BY created_at DESC LIMIT 1").Scan(&action)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return action, nil
}

func main() {
	// Парсим аргументы командной строки
	startButtonPtr := flag.Int("start", 1, "Начальная кнопка (1-6)")
	startItemPtr := flag.Int("item", 1, "Начальный предмет (1 для начала с первого)")
	flag.Parse()

	var startButton, startItem int

	// Если указан аргумент -start, используем его, иначе запрашиваем ввод через консоль
	if flag.NFlag() > 0 {
		// Проверяем валидность начальной кнопки из аргументов
		if *startButtonPtr < 1 || *startButtonPtr > 6 {
			log.Fatal("Начальная кнопка должна быть в диапазоне 1-6")
		}
		startButton = *startButtonPtr
		startItem = *startItemPtr
	} else {
		// Если аргумент не указан, запрашиваем ввод через консоль
		startButton = getStartButtonFromConsole()
		startItem = getStartItemFromConsole()
	}

	// init конфигурации
	err, c := config.InitConfig()
	if err != nil {
		return
	}

	// Устанавливаем начальную кнопку и предмет
	c.StartButtonIndex = startButton
	c.StartItemIndex = startItem

	// Инициализация логгера
	loggerManager, err := logger.NewLoggerManager(c.LogFilePath)
	if err != nil {
		log.Fatal("Error initializing logger: ", err)
	}
	defer loggerManager.Close()

	loggerManager.Info("🚀 Запуск приложения ШНЫРЬ")
	loggerManager.Info("🔘 Начальная кнопка: %d", c.StartButtonIndex)
	loggerManager.Info("📍 Начальный предмет: %d", c.StartItemIndex)

	// Подключение к базе данных MySQL
	db, err := sql.Open("mysql", "root:root@tcp(108.181.194.102:3306)/octopus?parseTime=true")
	if err != nil {
		loggerManager.LogError(err, "Error connecting to database")
		return
	}
	defer db.Close()

	// Проверяем подключение к базе данных
	err = db.Ping()
	if err != nil {
		loggerManager.LogError(err, "Error pinging database")
		return
	}
	loggerManager.Info("✅ Успешное подключение к базе данных")

	// Обновляем статус при запуске
	err = updateStatus(db, "main")
	if err != nil {
		loggerManager.LogError(err, "Error updating status")
	}
	err = addAction(db, "Приложение запущено")
	if err != nil {
		loggerManager.LogError(err, "Error adding action")
	}

	// Устанавливаем базу данных в пакете screenshot
	screenshot.SetDatabase(db)

	// Инициализация порта с использованием значений из конфигурации
	portObj, err := arduino.InitializePort(c.Port, c.BaudRate)
	if err != nil {
		loggerManager.LogError(err, "Error opening arduino port")
		return
	}
	defer func(port *serial.Port) {
		err := port.Close()
		if err != nil {
			loggerManager.LogError(err, "Error closing port")
		}
	}(portObj)

	// Устанавливаем объект порта в конфиг
	c.PortObj = portObj

	// Инициализация окна для получения отступов
	windowInitializer := imageInternal.NewWindowInitializer(c.WindowTopOffset)
	marginX, marginY, err := windowInitializer.GetItemBrokerWindowMargins()
	if err != nil {
		loggerManager.LogError(err, "Ошибка инициализации окна")
		return
	}

	// Инициализация всех менеджеров
	screenshotManager := screenshot.NewScreenshotManager(marginX, marginY)
	dbManager := database.NewDatabaseManager(db, loggerManager)
	ocrManager := ocr.NewOCRManager(&c)
	clickManager := click_manager.NewClickManager(portObj, &c, marginX, marginY, screenshotManager, dbManager, loggerManager)

	// Инициализация менеджера прерываний
	interruptManager := interrupt.NewInterruptManager(loggerManager)

	// Обработка завершения программы (после создания dbManager)
	defer func() {
		err = dbManager.UpdateStatus("stopped")
		if err != nil {
			loggerManager.LogError(err, "Error updating status to stopped on exit")
		}
		err = addAction(db, "Программа завершена")
		if err != nil {
			loggerManager.LogError(err, "Error adding exit action")
		}
		loggerManager.Info("🛑 Программа завершена")
	}()

	loggerManager.Info("⏸️ Программа готова к работе")
	loggerManager.Info("🔥 Горячие клавиши: Ctrl+Shift+1 для cycle_all_items, Ctrl+Shift+2 для cycle_listed_items, Q для прерывания")

	// Обновляем статус на "ready"
	err = updateStatus(db, "ready")
	if err != nil {
		loggerManager.LogError(err, "Error updating status to ready")
	}
	err = addAction(db, "Программа готова к работе")
	if err != nil {
		loggerManager.LogError(err, "Error adding ready action")
	}

	// запускаем мониторинг горячих клавиш
	interruptManager.StartMonitoring()

	// Запускаем горутину для периодической проверки невыполненных действий
	go func() {
		for {
			// Проверяем каждые 5 секунд
			time.Sleep(5 * time.Second)

			// Проверяем, не запущен ли уже скрипт
			if interruptManager.IsScriptRunning() {
				continue
			}

			// Проверяем последнее невыполненное действие
			latestPendingAction, err := getLatestPendingAction(db)
			if err != nil {
				loggerManager.LogError(err, "Ошибка получения последнего невыполненного действия")
				continue
			}

			if latestPendingAction == "start" {
				loggerManager.Info("🚀 Обнаружено невыполненное действие 'start', запускаем cycle_listed_items")

				// Помечаем действие как выполненное
				err = updateLatestPendingAction(db)
				if err != nil {
					loggerManager.LogError(err, "Ошибка обновления последнего действия")
				}

				// Запускаем cycle_listed_items
				err = dbManager.UpdateStatus("cycle_listed_items")
				if err != nil {
					loggerManager.LogError(err, "Error updating status to cycle_listed_items")
				}
				err = addAction(db, "Запуск cycle_listed_items (автоматический)")
				if err != nil {
					loggerManager.LogError(err, "Error adding cycle_listed_items action")
				}

				// Канал для завершения cycle_listed_items
				scriptDoneChan := make(chan bool, 1)
				interruptManager.SetScriptRunning(true)

				// Запускаем cycle_listed_items в отдельной горутине
				go func() {
					defer func() {
						// При завершении (нормальном или прерывании) обновляем статус
						if interruptManager.IsInterrupted() {
							err = dbManager.UpdateStatus("stopped")
							if err != nil {
								loggerManager.LogError(err, "Error updating status to stopped")
							}
							err = updateLatestPendingAction(db)
							if err != nil {
								loggerManager.LogError(err, "Error updating latest pending action")
							}
							err = addAction(db, "cycle_listed_items прерван")
							if err != nil {
								loggerManager.LogError(err, "Error adding interruption action")
							}
						} else {
							err = dbManager.UpdateStatus("ready")
							if err != nil {
								loggerManager.LogError(err, "Error updating status to ready")
							}
							err = updateLatestPendingAction(db)
							if err != nil {
								loggerManager.LogError(err, "Error updating latest pending action")
							}
							err = addAction(db, "cycle_listed_items завершен")
							if err != nil {
								loggerManager.LogError(err, "Error adding completion action")
							}
						}
						scriptDoneChan <- true
					}()

					cycleListedItems.Run(&c, screenshotManager, dbManager, ocrManager, clickManager, loggerManager, interruptManager)
				}()

				// Ждем завершения cycle_listed_items
				<-scriptDoneChan
				interruptManager.SetScriptRunning(false)
				loggerManager.Info("✅ cycle_listed_items завершен. Нажмите Ctrl+Shift+2 для повторного запуска")
			}
		}
	}()

	for range interruptManager.GetScriptStartChan() {
		// Сбрасываем флаг прерывания при запуске нового скрипта
		interruptManager.SetInterrupted(false)

		// Проверяем текущий статус перед запуском скрипта
		currentStatus, err := dbManager.GetCurrentStatus()
		if err != nil {
			loggerManager.LogError(err, "Ошибка получения текущего статуса")
			continue
		}

		// Запускаем скрипт только если статус "stopped", "ready" или "main"
		if currentStatus != "stopped" && currentStatus != "ready" && currentStatus != "main" {
			loggerManager.Info("⚠️ Скрипт не может быть запущен. Текущий статус: %s. Ожидаемый статус: stopped, ready или main", currentStatus)
			continue
		}

		// Определяем какой скрипт запускать по типу сигнала
		scriptType := interruptManager.GetLastScriptType()

		switch scriptType {
		case "cycle_all_items":
			loggerManager.Info("🚀 Запуск cycle_all_items...")
			loggerManager.Info("💡 Для прерывания нажмите Q (работает глобально)")

			// Обновляем статус на запуск скрипта
			err = dbManager.UpdateStatus("cycle_all_items")
			if err != nil {
				loggerManager.LogError(err, "Error updating status to cycle_all_items")
			}
			err = addAction(db, "Запуск cycle_all_items")
			if err != nil {
				loggerManager.LogError(err, "Error adding cycle_all_items action")
			}

			// Канал для завершения cycle_all_items
			scriptDoneChan := make(chan bool, 1)
			interruptManager.SetScriptRunning(true)

			// Запускаем cycle_all_items в отдельной горутине
			go func() {
				defer func() {
					// При завершении (нормальном или прерывании) обновляем статус
					if interruptManager.IsInterrupted() {
						err = dbManager.UpdateStatus("stopped")
						if err != nil {
							loggerManager.LogError(err, "Error updating status to stopped")
						}
						err = updateLatestPendingAction(db)
						if err != nil {
							loggerManager.LogError(err, "Error updating latest pending action")
						}
						err = addAction(db, "cycle_all_items прерван")
						if err != nil {
							loggerManager.LogError(err, "Error adding interruption action")
						}
					} else {
						err = dbManager.UpdateStatus("ready")
						if err != nil {
							loggerManager.LogError(err, "Error updating status to ready")
						}
						err = updateLatestPendingAction(db)
						if err != nil {
							loggerManager.LogError(err, "Error updating latest pending action")
						}
						err = addAction(db, "cycle_all_items завершен")
						if err != nil {
							loggerManager.LogError(err, "Error adding completion action")
						}
					}
					scriptDoneChan <- true
				}()

				cycleAllItems.Run(&c, screenshotManager, dbManager, ocrManager, clickManager, loggerManager, interruptManager)
			}()

			// Ждем завершения cycle_all_items
			<-scriptDoneChan
			interruptManager.SetScriptRunning(false)
			loggerManager.Info("✅ cycle_all_items завершен. Нажмите Ctrl+Shift+1 для повторного запуска")

		case "cycle_listed_items":
			loggerManager.Info("🚀 Запуск cycle_listed_items...")
			loggerManager.Info("💡 Для прерывания нажмите Q (работает глобально)")

			// Обновляем статус на запуск скрипта
			err = dbManager.UpdateStatus("cycle_listed_items")
			if err != nil {
				loggerManager.LogError(err, "Error updating status to cycle_listed_items")
			}
			err = addAction(db, "Запуск cycle_listed_items")
			if err != nil {
				loggerManager.LogError(err, "Error adding cycle_listed_items action")
			}

			// Канал для завершения cycle_listed_items
			scriptDoneChan := make(chan bool, 1)
			interruptManager.SetScriptRunning(true)

			// Запускаем cycle_listed_items в отдельной горутине
			go func() {
				defer func() {
					// При завершении (нормальном или прерывании) обновляем статус
					if interruptManager.IsInterrupted() {
						err = dbManager.UpdateStatus("stopped")
						if err != nil {
							loggerManager.LogError(err, "Error updating status to stopped")
						}
						err = updateLatestPendingAction(db)
						if err != nil {
							loggerManager.LogError(err, "Error updating latest pending action")
						}
						err = addAction(db, "cycle_listed_items прерван")
						if err != nil {
							loggerManager.LogError(err, "Error adding interruption action")
						}
					} else {
						err = dbManager.UpdateStatus("ready")
						if err != nil {
							loggerManager.LogError(err, "Error updating status to ready")
						}
						err = updateLatestPendingAction(db)
						if err != nil {
							loggerManager.LogError(err, "Error updating latest pending action")
						}
						err = addAction(db, "cycle_listed_items завершен")
						if err != nil {
							loggerManager.LogError(err, "Error adding completion action")
						}
					}
					scriptDoneChan <- true
				}()

				cycleListedItems.Run(&c, screenshotManager, dbManager, ocrManager, clickManager, loggerManager, interruptManager)
			}()

			// Ждем завершения cycle_listed_items
			<-scriptDoneChan
			interruptManager.SetScriptRunning(false)
			loggerManager.Info("✅ cycle_listed_items завершен. Нажмите Ctrl+Shift+2 для повторного запуска")
		}
	}
}
