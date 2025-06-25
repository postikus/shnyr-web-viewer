package arduino

import (
	"image"
	"octopus/internal/config"

	"github.com/tarm/serial"
)

var sendFastClickToArduino = func(x int, y int, port *serial.Port) {
	SendFastClickToArduino(port)
}

var sendScrollDownToArduino = func(x int, y int, port *serial.Port) {
	SendScrollDownToArduino(port, x)
}

var sendScrollUpToArduino = func(x int, y int, port *serial.Port) {
	SendScrollUpToArduino(port, x)
}

var sendCoordinatesToArduino = func(x int, y int, port *serial.Port) {
	SendCoordinatesToArduino(port, x, y)
}

// Wait for Arduino's response
var waitForArduinoResponse = func(expectedResponse string, port *serial.Port) (string, error) {
	return WaitForArduinoResponse(port, expectedResponse)
}

var FastClick = func(port *serial.Port, config *config.Config) {
	err := ProcessAndWait(sendFastClickToArduino, waitForArduinoResponse, nil, 0, 0, port, config)
	if err != nil {
		return
	}
}

var ClickCoordinates = func(port *serial.Port, config *config.Config, coordinates image.Point) {
	err := ProcessAndWait(sendCoordinatesToArduino, waitForArduinoResponse, nil, coordinates.X, coordinates.Y, port, config)
	if err != nil {
		return
	}
}

var ScrollDown = func(port *serial.Port, config *config.Config, x int) {
	err := ProcessAndWait(sendScrollDownToArduino, waitForArduinoResponse, nil, x, 0, port, config)
	if err != nil {
		return
	}
}

var ScrollUp = func(port *serial.Port, config *config.Config, x int) {
	err := ProcessAndWait(sendScrollUpToArduino, waitForArduinoResponse, nil, x, 0, port, config)
	if err != nil {
		return
	}
}
