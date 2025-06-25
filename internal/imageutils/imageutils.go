package imageutils

import (
	"fmt"
	"image"
	"image/draw"
)

// CombineImages объединяет изображения
func CombineImages(imgs []image.Image, extraImg image.Image, extraOffset int) (*image.RGBA, error) {
	offset := 0
	width := imgs[0].Bounds().Dx()
	height := imgs[0].Bounds().Dy()

	combinedImg := image.NewRGBA(image.Rect(0, 0, width, 1000))
	draw.Draw(combinedImg, image.Rect(0, 0, width, height), imgs[0], image.Point{}, draw.Over)

	if len(imgs) > 1 {
		for _, img := range imgs[1 : len(imgs)-1] {
			offset += 30
			draw.Draw(combinedImg, image.Rect(0, offset, width, height+offset), img, image.Point{}, draw.Over)
		}
	}

	// Добавляем extraImg с extraOffset, если он не nil
	if extraImg != nil {
		offset += extraOffset
		draw.Draw(combinedImg, image.Rect(0, offset, width, height+offset), extraImg, image.Point{}, draw.Over)
	}

	return combinedImg, nil
}

// CropOpacityPixel обрезает изображение по непрозрачным пикселям
func CropOpacityPixel(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	top := height
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha > 0 {
				top = y + 1
				break
			}
		}
		if top != height {
			break
		}
	}
	return img.(*image.RGBA).SubImage(image.Rect(0, 0, width, top))
}

// LastColorStripeY ищет последнюю (нижнюю) горизонтальную полоску, где подряд минимум minLen пикселей с red > minR
func LastColorStripeY(img image.Image, minR int, minLen int) (int, error) {
	bounds := img.Bounds()
	for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
		count := 0
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, _, _, _ := GetPixelColor(img, x, y)
			if r > minR {
				count++
				if count >= minLen {
					return y, nil
				}
			} else {
				count = 0
			}
		}
	}
	return -1, fmt.Errorf("Не найдена горизонтальная полоска (red > %d, minLen=%d)", minR, minLen)
}

// LastColorStripeDistanceDiff возвращает разницу расстояний до последней цветной полоски для двух картинок
func LastColorStripeDistanceDiff(img1, img2 image.Image, minR int, minLen int) (int, error) {
	y1, err1 := LastColorStripeY(img1, minR, minLen)
	y2, err2 := LastColorStripeY(img2, minR, minLen)
	if err1 != nil || err2 != nil {
		return 0, fmt.Errorf("Ошибка поиска полоски: %v, %v", err1, err2)
	}
	return y1 - y2, nil
}

// GetPixelColor копия из internal/image/image.go
func GetPixelColor(img image.Image, x int, y int) (int, int, int, error) {
	color := img.At(x, y)
	r, g, b, _ := color.RGBA()
	rDecimal := int(r >> 8)
	gDecimal := int(g >> 8)
	bDecimal := int(b >> 8)
	return rDecimal, gDecimal, bDecimal, nil
}
