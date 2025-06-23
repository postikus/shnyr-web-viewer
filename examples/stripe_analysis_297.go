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

// Ищет самый длинный непрерывный диапазон, где red > minR, и возвращает его границы
func FindLongestStripeRedGreaterThan(img image.Image, x, minR int) (int, int, error) {
	bounds := img.Bounds()
	maxStart, maxEnd, maxLen := -1, -1, 0
	curStart, curLen := -1, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		r, _, _, _ := customimage.GetPixelColor(img, x, y)
		if r > minR {
			if curStart == -1 {
				curStart = y
				curLen = 1
			} else {
				curLen++
			}
			if curLen > maxLen {
				maxLen = curLen
				maxStart = curStart
				maxEnd = y
			}
		} else {
			curStart = -1
			curLen = 0
		}
	}
	if maxStart == -1 || maxEnd == -1 {
		return -1, -1, fmt.Errorf("Не найден непрерывный диапазон с red > %d", minR)
	}
	return maxStart, maxEnd, nil
}

// Возвращает расстояние до полоски (startY)
func analyzeImage(filePath string, outputFilePath string, targetX int, minR int) int {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Не удалось открыть файл изображения '%s': %v\n", filePath, err)
		return -1
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Не удалось декодировать изображение '%s': %v\n", filePath, err)
		return -1
	}

	bounds := img.Bounds()
	fmt.Printf("\nФайл: %s\n", filePath)
	fmt.Printf("Размеры изображения: %dx%d\n", bounds.Dx(), bounds.Dy())

	if targetX < bounds.Min.X || targetX >= bounds.Max.X {
		log.Printf("Координата X=%d вне изображения (0-%d)\n", targetX, bounds.Max.X-1)
		return -1
	}

	startY, endY, err := FindLongestStripeRedGreaterThan(img, targetX, minR)
	if err != nil {
		fmt.Println(err)
		return -1
	}
	distance := startY
	fmt.Printf("Расстояние от верхнего края до начала полоски по X = %d: %d пикселей\n", targetX, distance)
	fmt.Printf("Начало полоски: Y = %d\n", startY)
	fmt.Printf("Конец полоски: Y = %d\n", endY)
	fmt.Printf("Длина полоски: %d пикселей\n", endY-startY+1)

	imgWithAnalysis := image.NewRGBA(bounds)
	draw.Draw(imgWithAnalysis, bounds, img, image.Point{}, draw.Src)

	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	for y := 0; y < startY; y++ {
		imgWithAnalysis.Set(targetX, y, blue)
	}
	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	for y := startY; y <= endY; y++ {
		imgWithAnalysis.Set(targetX, y, redColor)
	}

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		log.Printf("Не удалось создать выходной файл изображения: %v\n", err)
		return distance
	}
	defer outputFile.Close()

	err = png.Encode(outputFile, imgWithAnalysis)
	if err != nil {
		log.Printf("Не удалось закодировать изображение: %v\n", err)
		return distance
	}

	fmt.Printf("Сохранено изображение с анализом в '%s'\n", outputFilePath)
	fmt.Printf("Синяя линия - расстояние до полоски, красная линия - сама полоска (X=%d, Y=%d..%d)\n", targetX, startY, endY)
	return distance
}

func main() {
	dist1 := analyzeImage("test_image.png", "stripe_analysis_x297_1.png", 297, 26)
	dist2 := analyzeImage("test_image2.png", "stripe_analysis_x297_2.png", 297, 26)
	if dist1 >= 0 && dist2 >= 0 {
		delta := dist2 - dist1
		fmt.Printf("\nРазница между расстояниями до полоски: %d (test_image2 - test_image1)\n", delta)
	}
}
