package main

import (
	"database/sql"
	"log"
	"octopus/internal/arduino"
	"octopus/internal/click_manager"
	"octopus/internal/config"
	"octopus/internal/database"
	imageInternal "octopus/internal/image"
	"octopus/internal/logger"
	"octopus/internal/ocr"
	"octopus/internal/screenshot"
	script1 "octopus/internal/scripts/script1"

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

	loggerManager.Info("🚀 Запуск приложения Octopus")

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

	// Запуск скрипта с переданными менеджерами
	script1.Run(&c, screenshotManager, dbManager, ocrManager, clickManager, marginX, marginY, loggerManager)
}
