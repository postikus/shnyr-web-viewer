package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	imageInternal "octopus/internal/image"
	"os"
)

func main() {
	// Создаем простое тестовое изображение
	width, height := 800, 600
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Заполняем черным фоном
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	// Создаем окно размером 400x500 (как в алгоритме)
	windowX, windowY := 200, 50
	windowWidth, windowHeight := 400, 500

	// Заполняем окно белым цветом (очень заметно)
	for y := windowY; y < windowY+windowHeight; y++ {
		for x := windowX; x < windowX+windowWidth; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	// Сохраняем тестовое изображение
	testFile, err := os.Create("debug_screen.png")
	if err != nil {
		log.Fatal(err)
	}
	defer testFile.Close()
	png.Encode(testFile, img)
	fmt.Println("Отладочное изображение сохранено как debug_screen.png")

	// Проверяем несколько пикселей
	fmt.Println("Проверка пикселей:")
	fmt.Printf("Фон (0,0): %v\n", img.At(0, 0))
	fmt.Printf("Окно (250,100): %v\n", img.At(250, 100))
	fmt.Printf("Окно (300,200): %v\n", img.At(300, 200))

	// Ищем окно
	gameWindow, err := imageInternal.FindGameWindow(img)
	if err != nil {
		fmt.Printf("Ошибка поиска окна: %v\n", err)
		return
	}

	fmt.Printf("Окно найдено!\n")
	fmt.Printf("Координаты: X=%d, Y=%d\n", gameWindow.X, gameWindow.Y)
	fmt.Printf("Размеры: Width=%d, Height=%d\n", gameWindow.Width, gameWindow.Height)
}
