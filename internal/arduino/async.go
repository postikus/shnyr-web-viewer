package arduino

import (
	"fmt"
	"shnyr/internal/config"

	"github.com/tarm/serial"
)

// ProcessAndWait выполняет отправку координат, выполнение колбэка и ожидание ответа от Arduino
func ProcessAndWait(
	sendCoordinatesToArduino func(int, int, *serial.Port), // Принимаем указатели на int для координат
	waitForArduinoResponse func(string, *serial.Port) (string, error),
	callback func(config *config.Config), // Колбэк для выполнения любых действий с любым типом данных
	x, y int, config *config.Config) error { // Передаем координаты клика как указатели

	// Если координаты переданы (не nil), отправляем их в Arduino
	sendCoordinatesToArduino(x, y, config.PortObj)

	// Ожидаем ответа от Arduino после выполнения колбэка
	_, err := waitForArduinoResponse("received", config.PortObj)
	if err != nil {
		return fmt.Errorf("error waiting for Arduino response: %v", err)
	}

	// Выполняем колбэк, если он не nil
	if callback != nil {
		callback(config)
	}

	return nil
}
