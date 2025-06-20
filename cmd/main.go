package main

import (
	"github.com/tarm/serial"
	"log"
	"octopus/internal/arduino"
	"octopus/internal/config"
	script1 "octopus/internal/scripts/script1"
)

func main() {
	// init конфигурации
	err, c := config.InitConfig()
	if err != nil {
		return
	}

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

	// Запуск скрипта с переданным портом
	script1.Run(portObj, &c)
}
