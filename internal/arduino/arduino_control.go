package arduino

import (
	"image"
	"shnyr/internal/config"

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

var sendKeyDownToArduino = func(key string, y int, port *serial.Port) {
	SendKeyDownToArduino(port, key)
}

var sendKeyUpToArduino = func(key string, y int, port *serial.Port) {
	SendKeyUpToArduino(port, key)
}

// Wait for Arduino's response
var waitForArduinoResponse = func(expectedResponse string, port *serial.Port) (string, error) {
	return WaitForArduinoResponse(port, expectedResponse)
}

var FastClick = func(config *config.Config) {
	err := ProcessAndWait(sendFastClickToArduino, waitForArduinoResponse, nil, 0, 0, config)
	if err != nil {
		return
	}
}

var ClickCoordinates = func(config *config.Config, coordinates image.Point) {
	err := ProcessAndWait(sendCoordinatesToArduino, waitForArduinoResponse, nil, coordinates.X, coordinates.Y, config)
	if err != nil {
		return
	}
}

var ScrollDown = func(config *config.Config, x int) {
	err := ProcessAndWait(sendScrollDownToArduino, waitForArduinoResponse, nil, x, 0, config)
	if err != nil {
		return
	}
}

var ScrollUp = func(config *config.Config, x int) {
	err := ProcessAndWait(sendScrollUpToArduino, waitForArduinoResponse, nil, x, 0, config)
	if err != nil {
		return
	}
}

var KeyDown = func(config *config.Config, key string) {
	SendKeyDownToArduino(config.PortObj, key)
	_, err := waitForArduinoResponse("received", config.PortObj)
	if err != nil {
		return
	}
}

var KeyUp = func(config *config.Config, key string) {
	SendKeyUpToArduino(config.PortObj, key)
	_, err := waitForArduinoResponse("received", config.PortObj)
	if err != nil {
		return
	}
}

var Paste = func(config *config.Config) {
	SendPasteToArduino(config.PortObj)
	_, err := waitForArduinoResponse("received", config.PortObj)
	if err != nil {
		return
	}
}

var F12 = func(config *config.Config) {
	SendF12ToArduino(config.PortObj)
	_, err := waitForArduinoResponse("received", config.PortObj)
	if err != nil {
		return
	}
}

var CopyToClipboard = func(config *config.Config, text string) {
	SendTextToClipboard(config.PortObj, text)
	_, err := waitForArduinoResponse("received", config.PortObj)
	if err != nil {
		return
	}
}
