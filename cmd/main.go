package main

import (
	"database/sql"
	"log"
	"octopus/internal/arduino"
	"octopus/internal/click_manager"
	"octopus/internal/config"
	"octopus/internal/database"
	imageInternal "octopus/internal/image"
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

	// Подключение к базе данных MySQL
	db, err := sql.Open("mysql", "root:root@tcp(108.181.194.102:3306)/octopus?parseTime=true")
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}
	defer db.Close()

	// Проверяем подключение к базе данных
	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging database: ", err)
	}
	log.Println("Successfully connected to database")

	// Устанавливаем базу данных в пакете screenshot
	screenshot.SetDatabase(db)

	// Инициализация порта с использованием значений из конфигурации
	portObj, err := arduino.InitializePort(c.Port, c.BaudRate)
	if err != nil {
		log.Fatal("Error opening arduino port: ", err)
	}
	defer func(port *serial.Port) {
		err := port.Close()
		if err != nil {
			log.Println("Error closing port:", err)
		}
	}(portObj)

	// Инициализация окна для получения отступов
	windowInitializer := imageInternal.NewWindowInitializer(c.WindowTopOffset)
	marginX, marginY, err := windowInitializer.GetItemBrokerWindowMargins()
	if err != nil {
		log.Fatalf("Ошибка инициализации окна: %v", err)
	}

	// Инициализация всех менеджеров
	screenshotManager := screenshot.NewScreenshotManager(marginX, marginY)
	dbManager := database.NewDatabaseManager(db)
	ocrManager := ocr.NewOCRManager(&c)
	clickManager := click_manager.NewClickManager(portObj, &c, marginX, marginY, screenshotManager, dbManager)

	// Запуск скрипта с переданными менеджерами
	script1.Run(&c, screenshotManager, dbManager, ocrManager, clickManager, marginX, marginY)
}
