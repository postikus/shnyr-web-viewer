package main

import (
	"database/sql"
	"log"
	"shnyr/internal/arduino"
	"shnyr/internal/click_manager"
	"shnyr/internal/config"
	"shnyr/internal/database"
	imageInternal "shnyr/internal/image"
	"shnyr/internal/interrupt"
	"shnyr/internal/logger"
	"shnyr/internal/ocr"
	"shnyr/internal/screenshot"
	script1 "shnyr/internal/scripts/script1"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tarm/serial"
)

func main() {
	// init конфигурации
	err, c := config.InitConfig()
	if err != nil {
		return
	}

	// Инициализация логгера
	loggerManager, err := logger.NewLoggerManager(c.LogFilePath)
	if err != nil {
		log.Fatal("Error initializing logger: ", err)
	}
	defer loggerManager.Close()

	loggerManager.Info("🚀 Запуск приложения ШНЫРЬ")

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
	loggerManager.Info("⏸️ Программа готова к работе. Нажмите Shift+Enter для запуска script1, Q для прерывания")
	loggerManager.Info("🔥 Горячие клавиши: Shift+Enter для запуска, Q для прерывания script1")

	// запускаем мониторинг горячих клавиш
	interruptManager.StartMonitoring()

	for range interruptManager.GetScriptStartChan() {
		loggerManager.Info("🚀 Запуск script1...")
		loggerManager.Info("💡 Для прерывания нажмите Q (работает глобально)")

		// Канал для завершения script1
		scriptDoneChan := make(chan bool, 1)
		interruptManager.SetScriptRunning(true)

		// Запускаем script1 в отдельной горутине
		go func() {
			script1.Run(&c, screenshotManager, dbManager, ocrManager, clickManager, loggerManager, interruptManager)
			scriptDoneChan <- true
		}()

		// Ждем завершения script1
		<-scriptDoneChan
		interruptManager.SetScriptRunning(false)
		loggerManager.Info("✅ script1 завершен. Нажмите Shift+Enter для повторного запуска")
	}
}
