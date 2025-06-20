package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	imageInternal "octopus/internal/image"
	"octopus/internal/screen"
	"os"
)

func main() {
	// Количество пикселей для отрезания сверху
	topOffset := 23

	// Делаем скриншот всего экрана
	img, err := screen.CaptureFullScreen()
	if err != nil {
		log.Fatalf("Ошибка захвата экрана: %v", err)
	}

	// Обрезаем верхние пиксели
	bounds := img.Bounds()
	croppedImg := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()-topOffset))
	for y := topOffset; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			croppedImg.Set(x, y-topOffset, img.At(x, y))
		}
	}

	// Ищем окно
	gameWindow, err := imageInternal.FindGameWindow(croppedImg)
	if err != nil {
		fmt.Printf("Окно не найдено: %v\n", err)
		return
	}

	// --- Новый скриншот и вырезание области 400x500 ---
	cropX := gameWindow.X - 150
	cropY := gameWindow.Y + topOffset + 45
	cropWidth := 300
	cropHeight := 364

	imgFull, err := screen.CaptureFullScreen()
	if err != nil {
		log.Fatalf("Ошибка захвата экрана: %v", err)
	}

	cropped := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	for y := 0; y < cropHeight; y++ {
		for x := 0; x < cropWidth; x++ {
			cropped.Set(x, y, imgFull.At(cropX+x, cropY+y))
		}
	}

	cropFile, err := os.Create("cropped_window.png")
	if err != nil {
		log.Fatalf("Ошибка создания файла: %v", err)
	}
	defer cropFile.Close()
	png.Encode(cropFile, cropped)
	fmt.Println("Вырезанный скриншот сохранён как cropped_window.png")
}
