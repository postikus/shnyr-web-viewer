package scripts

import (
	"github.com/tarm/serial"
	"octopus/internal/arduino"
	"octopus/internal/config"
)

var sendFastClickToArduino = func(x int, y int, port *serial.Port) {
	arduino.SendFastClickToArduino(port)
}

var sendScrollDownToArduino = func(x int, y int, port *serial.Port) {
	arduino.SendScrollDownToArduino(port, x)
}

var sendScrollUpToArduino = func(x int, y int, port *serial.Port) {
	arduino.SendScrollUpToArduino(port, x)
}

var sendCoordinatesToArduino = func(x int, y int, port *serial.Port) {
	arduino.SendCoordinatesToArduino(port, x, y)
}

// Wait for Arduino's response
var waitForArduinoResponse = func(expectedResponse string, port *serial.Port) (string, error) {
	return arduino.WaitForArduinoResponse(port, expectedResponse)
}

var FastClick = func(port *serial.Port, config *config.Config) {
	err := arduino.ProcessAndWait(sendFastClickToArduino, waitForArduinoResponse, nil, 0, 0, port, config)
	if err != nil {
		return
	}
}

var ClickCoordinates = func(port *serial.Port, config *config.Config, coordinates config.Coordinates) {
	err := arduino.ProcessAndWait(sendCoordinatesToArduino, waitForArduinoResponse, nil, coordinates.X, coordinates.Y, port, config)
	if err != nil {
		return
	}
}

var ScrollDown = func(port *serial.Port, config *config.Config, x int) {
	err := arduino.ProcessAndWait(sendScrollDownToArduino, waitForArduinoResponse, nil, x, 0, port, config)
	if err != nil {
		return
	}
}

var ScrollUp = func(port *serial.Port, config *config.Config, x int) {
	err := arduino.ProcessAndWait(sendScrollUpToArduino, waitForArduinoResponse, nil, x, 0, port, config)
	if err != nil {
		return
	}
}
