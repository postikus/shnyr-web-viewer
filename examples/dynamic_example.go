package main

import (
	"log"
	"octopus/internal/arduino"
	"octopus/internal/config"

	"github.com/tarm/serial"
)

func main() {
	// init конфигурации
	err, c := config.InitConfig()
	if err != nil {
		return
	}

	// Создаем динамическую конфигурацию
	dynamicConfig := config.NewDynamicConfig(&c)

	// Находим окно игры
	err = dynamicConfig.FindAndSetGameWindow()
	if err != nil {
		log.Printf("Warning: Could not find game window, using static coordinates: %v", err)
	} else {
		log.Printf("Game window found at: X=%d, Y=%d, Width=%d, Height=%d",
			dynamicConfig.GameWindow.X, dynamicConfig.GameWindow.Y,
			dynamicConfig.GameWindow.Width, dynamicConfig.GameWindow.Height)
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

	// Пример использования динамических координат
	// Получаем абсолютные координаты для клика по кнопке
	absX, absY := dynamicConfig.GetAbsoluteCoordinates(c.Click.Button1)
	log.Printf("Absolute coordinates for Button1: X=%d, Y=%d", absX, absY)

	// Получаем абсолютные координаты для скриншота
	absScreenshot := dynamicConfig.GetAbsoluteCoordinatesWithSize(c.Screenshot.ItemList)
	log.Printf("Absolute screenshot coordinates: X=%d, Y=%d, Width=%d, Height=%d",
		absScreenshot.X, absScreenshot.Y, absScreenshot.Width, absScreenshot.Height)
}
