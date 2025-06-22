package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"

	customimage "octopus/internal/image"
)

func main() {
	// 1. Загружаем изображение
	filePath := "cropped_window.png"
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Не удалось открыть файл изображения '%s': %v", filePath, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatalf("Не удалось декодировать изображение: %v", err)
	}

	// 2. Находим позиции предметов по цвету текста, передавая желаемую координату X.
	itemPositions := customimage.FindItemPositionsByTextColor(img, 80)

	// 3. Выводим координаты
	fmt.Printf("Найдено %d позиций предметов.\n", len(itemPositions))
	for i, center := range itemPositions {
		fmt.Printf("Позиция %d: X=%d, Y=%d\n", i+1, center.X, center.Y)
	}

	// 4. Создаем новое изображение для отрисовки результатов для проверки
	// Нам нужно изменяемое изображение, поэтому мы копируем исходное в новое изображение RGBA.
	bounds := img.Bounds()
	imgWithMarkers := image.NewRGBA(bounds)
	draw.Draw(imgWithMarkers, bounds, img, image.Point{}, draw.Src)

	// Рисуем красный крест в каждом центре
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	markerSize := 5 // Размер лучей креста

	for _, center := range itemPositions {
		// Горизонтальная линия
		for x := center.X - markerSize; x <= center.X+markerSize; x++ {
			if x >= bounds.Min.X && x < bounds.Max.X {
				imgWithMarkers.Set(x, center.Y, red)
			}
		}
		// Вертикальная линия
		for y := center.Y - markerSize; y <= center.Y+markerSize; y++ {
			if y >= bounds.Min.Y && y < bounds.Max.Y {
				imgWithMarkers.Set(center.X, y, red)
			}
		}
	}

	// 5. Сохраняем новое изображение
	outputFilePath := "items_located.png"
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		log.Fatalf("Не удалось создать выходной файл изображения: %v", err)
	}
	defer outputFile.Close()

	err = png.Encode(outputFile, imgWithMarkers)
	if err != nil {
		log.Fatalf("Не удалось закодировать изображение: %v", err)
	}

	fmt.Printf("Сохранено изображение с маркерами в '%s'\n", outputFilePath)
}
