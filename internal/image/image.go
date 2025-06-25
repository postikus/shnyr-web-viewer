package image

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
)

// Функция для проверки цвета пикселя по координатам
func GetPixelColor(img image.Image, x int, y int) (int, int, int, error) {
	color := img.At(x, y)

	// Получаем компоненты цвета (RGBA) и делим на 256, чтобы получить значения от 0 до 255
	r, g, b, _ := color.RGBA()

	// Преобразуем значения из диапазона 0-65535 в 0-255
	rDecimal := int(r >> 8)
	gDecimal := int(g >> 8)
	bDecimal := int(b >> 8)

	return rDecimal, gDecimal, bDecimal, nil
}

// Функция для объединения изображений
func CombineImages(imgs []image.Image, smallImgs []image.Image, extraImg image.Image, extraOffset int) (*image.RGBA, error) {
	offsetDif := 30
	offset := 0
	width := imgs[0].Bounds().Dx()
	height := imgs[0].Bounds().Dy()

	combinedImg := image.NewRGBA(image.Rect(0, 0, width, 1000))
	draw.Draw(combinedImg, imgs[0].Bounds(), imgs[0], image.Point{}, draw.Over)

	for _, img := range imgs[1:] {
		offset += offsetDif
		draw.Draw(combinedImg, image.Rect(0, offset, width, height+offset), img, image.Point{}, draw.Over)
	}

	offsetDif = 10
	for _, img := range smallImgs {
		offset += offsetDif
		draw.Draw(combinedImg, image.Rect(0, offset, width, height+offset), img, image.Point{}, draw.Over)
	}

	// Добавляем extraImg с extraOffset, если он не nil
	if extraImg != nil {
		offset += extraOffset
		draw.Draw(combinedImg, image.Rect(0, offset, width, height+offset), extraImg, image.Point{}, draw.Over)
	}

	return combinedImg, nil
}

// Функция для сохранения итогового изображения
var SaveCombinedImage = func(image image.Image, filename string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer outFile.Close()

	// Сохраняем изображение в PNG формате
	err = png.Encode(outFile, image)
	if err != nil {
		return fmt.Errorf("failed to save image: %v", err)
	}

	// TODO: Добавить прямую интеграцию OCR здесь
	// Вместо вызова exec.Command

	return nil
}

func CropOpacityPixel(img image.Image) image.Image {
	// Получаем размеры изображения
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Ищем, на какой строке снизу начинается непрозрачный пиксель
	top := height
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha > 0 { // Проверяем, что пиксель не прозрачный
				top = y + 1
				break
			}
		}
		if top != height {
			break
		}
	}

	// Обрезаем изображение до найденной строки
	return img.(*image.RGBA).SubImage(image.Rect(0, 0, width, top))
}

// ImageToBytes конвертирует изображение в байты в формате PNG
func ImageToBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования изображения: %v", err)
	}
	return buf.Bytes(), nil
}
