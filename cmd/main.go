package main

import (
	"database/sql"
	"flag"
	"log"
	"octopus/internal/arduino"
	"octopus/internal/config"
	"octopus/internal/screenshot"
	script1 "octopus/internal/scripts/script1"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tarm/serial"
)

// Глобальная переменная для отслеживания необходимости сохранения скриншотов
var SaveScreenshotsLocally bool

func main() {
	// Парсим аргументы командной строки
	saveScreenshots := flag.Bool("savess", false, "Save screenshots locally")
	flag.Parse()

	// Устанавливаем глобальную переменную
	SaveScreenshotsLocally = *saveScreenshots

	if SaveScreenshotsLocally {
		log.Println("📸 Локальное сохранение скриншотов ВКЛЮЧЕНО")
	} else {
		log.Println("📸 Локальное сохранение скриншотов ОТКЛЮЧЕНО")
	}

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

	// Устанавливаем флаг сохранения скриншотов локально
	screenshot.SetSaveScreenshotsLocally(SaveScreenshotsLocally)

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

	// Запуск скрипта с переданным портом и базой данных
	script1.Run(portObj, &c, db)
}
