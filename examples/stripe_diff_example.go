package main

import (
	"fmt"
	"image"
	"log"
	"os"

	customimage "shnyr/internal/image"
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

func stripeDiffMain() {
	img1 := loadImage("../last_prev.png")
	img2 := loadImage("../last.png")

	diff, err := customimage.LastColorStripeDistanceDiff(img1, img2, 26, 20)
	if err != nil {
		log.Fatalf("Ошибка при вычислении разницы: %v", err)
	}
	fmt.Printf("Разница по последней горизонтальной полоске (last - last_prev): %d\n", diff)
}
