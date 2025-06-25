package helpers

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

// GetPixelColor получает цвет пикселя по координатам
func GetPixelColor(img image.Image, x int, y int) (int, int, int, error) {
	bounds := img.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
		return 0, 0, 0, nil
	}

	clr := img.At(x, y)
	r, g, b, _ := clr.RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8), nil
}

// SaveCombinedImage сохраняет объединенное изображение в файл
func SaveCombinedImage(image image.Image, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	err = png.Encode(file, image)
	if err != nil {
		return fmt.Errorf("failed to encode image: %v", err)
	}

	return nil
}
