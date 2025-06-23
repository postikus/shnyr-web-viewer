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

// Ищет первый и последний пиксели с красным компонентом в диапазоне [minR, maxR] в столбце X
func FindStripeBoundsWithRedInColumn(img image.Image, x, minR, maxR int) (int, int, error) {
	bounds := img.Bounds()
	first, last := -1, -1
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		r, _, _, _ := customimage.GetPixelColor(img, x, y)
		if r >= minR && r <= maxR {
			if first == -1 {
				first = y
			}
			last = y
		}
	}
	if first == -1 || last == -1 {
		return -1, -1, fmt.Errorf("Не найдена полоска с R в диапазоне [%d, %d]", minR, maxR)
	}
	return first, last, nil
}

func main() {
	filePath := "test_image.png"
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Не удалось открыть файл изображения '%s': %v", filePath, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatalf("Не удалось декодировать изображение: %v", err)
	}

	bounds := img.Bounds()
	fmt.Printf("Размеры изображения: %dx%d\n", bounds.Dx(), bounds.Dy())

	targetX := 297
	fmt.Printf("Анализируем вертикальную полоску по координате X = %d\n", targetX)

	if targetX < bounds.Min.X || targetX >= bounds.Max.X {
		log.Fatalf("Координата X=%d находится за пределами изображения (0-%d)", targetX, bounds.Max.X-1)
	}

	minR, maxR := 25, 27
	startY, endY, err := FindStripeBoundsWithRedInColumn(img, targetX, minR, maxR)
	if err != nil {
		fmt.Println(err)
		return
	}

	stripeLength := endY - startY
	fmt.Printf("\nНайдена одна вертикальная полоска по X = %d:\n", targetX)
	fmt.Printf("Начало полоски: Y = %d\n", startY)
	fmt.Printf("Конец полоски: Y = %d\n", endY)
	fmt.Printf("Длина вертикальной полоски: %d пикселей\n", stripeLength)

	imgWithAnalysis := image.NewRGBA(bounds)
	draw.Draw(imgWithAnalysis, bounds, img, image.Point{}, draw.Src)

	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	for y := startY; y <= endY; y++ {
		imgWithAnalysis.Set(targetX, y, redColor)
	}

	outputFilePath := "stripe_analysis_x297.png"
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		log.Fatalf("Не удалось создать выходной файл изображения: %v", err)
	}
	defer outputFile.Close()

	err = png.Encode(outputFile, imgWithAnalysis)
	if err != nil {
		log.Fatalf("Не удалось закодировать изображение: %v", err)
	}

	fmt.Printf("\nСохранено изображение с анализом в '%s'\n", outputFilePath)
	fmt.Printf("Красная линия - найденная полоска (X=%d, Y=%d..%d)\n", targetX, startY, endY)
}
