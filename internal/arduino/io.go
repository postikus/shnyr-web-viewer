package arduino

import (
	"bytes"
	"fmt"

	"github.com/tarm/serial"
)

func InitializePort(name string, baud int) (*serial.Port, error) {
	port, err := serial.OpenPort(&serial.Config{
		Name:     name,
		Baud:     baud,
		Parity:   serial.ParityNone,
		StopBits: serial.Stop1,
	})
	return port, err
}

func SendFastClickToArduino(port *serial.Port) {
	message := fmt.Sprintf("fast_click")
	_, err := port.Write([]byte(message))
	if err != nil {
		fmt.Println("Error writing to Arduino:", err)
	}
}

func SendCoordinatesToArduino(port *serial.Port, x, y int) {
	message := fmt.Sprintf("click:%d,%d\n", x, y)
	_, err := port.Write([]byte(message))
	if err != nil {
		fmt.Println("Error writing to Arduino:", err)
	}
}

func SendScrollDownToArduino(port *serial.Port, x int) {
	message := fmt.Sprintf("scroll_down:%d\n", x)
	fmt.Println(message)
	_, err := port.Write([]byte(message))
	if err != nil {
		fmt.Println("Error writing to Arduino:", err)
	}
}

func SendScrollUpToArduino(port *serial.Port, x int) {
	message := fmt.Sprintf("scroll_up:%d\n", x)
	fmt.Println(message)
	_, err := port.Write([]byte(message))
	if err != nil {
		fmt.Println("Error writing to Arduino:", err)
	}
}

func WaitForArduinoResponse(port *serial.Port, expectedResponse string) (string, error) {
	var response string
	for {
		buf := make([]byte, 128)
		n, err := port.Read(buf)
		if err != nil {
			return "", fmt.Errorf("error reading from Arduino: %v", err)
		}

		response += string(buf[:n])

		if len(response) > 0 && response[len(response)-1] == '\n' {
			// Trim the newline character and any surrounding spaces
			response = response[:len(response)-1]
			response = string(bytes.TrimSpace([]byte(response))) // Trim whitespace

			if response == expectedResponse {
				return response, nil
			}
			return "", fmt.Errorf("unexpected response: '%s'", response)
		}
	}
}
