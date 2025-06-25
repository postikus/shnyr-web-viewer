package arduino

import (
	"fmt"
	"github.com/tarm/serial"
	"shnyr/internal/config"
)

// ProcessAndWait выполняет отправку координат, выполнение колбэка и ожидание ответа от Arduino
func ProcessAndWait(
	sendCoordinatesToArduino func(int, int, *serial.Port), // Принимаем указатели на int для координат
	waitForArduinoResponse func(string, *serial.Port) (string, error),
	callback func(config *config.Config), // Колбэк для выполнения любых действий с любым типом данных
	x, y int, port *serial.Port, config *config.Config) error { // Передаем координаты клика как указатели

	// Если координаты переданы (не nil), отправляем их в Arduino
	sendCoordinatesToArduino(x, y, port)

	// Ожидаем ответа от Arduino после выполнения колбэка
	_, err := waitForArduinoResponse("received", port)
	if err != nil {
		return fmt.Errorf("error waiting for Arduino response: %v", err)
	}

	// Выполняем колбэк, если он не nil
	if callback != nil {
		callback(config)
	}

	return nil
}
