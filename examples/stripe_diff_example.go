package main

import (
	"fmt"
	"image"
	"log"
	"os"

	customimage "octopus/internal/image"
)

func loadImage(path string) image.Image {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Не удалось открыть файл %s: %v", path, err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatalf("Не удалось декодировать изображение %s: %v", path, err)
	}
	return img
}

func main() {
	img1 := loadImage("last_prev.png")
	img2 := loadImage("last.png")

	diff, err := customimage.StripeDistanceDiff(img1, img2, 297, 26)
	if err != nil {
		log.Fatalf("Ошибка при вычислении разницы: %v", err)
	}
	fmt.Printf("Разница расстояний до полоски (last - last_prev): %d\n", diff)
}
