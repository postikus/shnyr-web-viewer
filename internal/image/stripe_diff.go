package image

import (
	"fmt"
	"image"
)

// FindLongestStripeRedGreaterThan ищет самый длинный непрерывный диапазон, где red > minR, и возвращает его границы
func FindLongestStripeRedGreaterThan(img image.Image, x, minR int) (int, int, error) {
	bounds := img.Bounds()
	maxStart, maxEnd, maxLen := -1, -1, 0
	curStart, curLen := -1, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		r, _, _, _ := GetPixelColor(img, x, y)
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

// StripeDistanceDiff возвращает разницу расстояний от верхнего края до начала полоски (red > minR) по X для двух изображений
func StripeDistanceDiff(img1, img2 image.Image, x, minR int) (int, error) {
	startY1, _, err1 := FindLongestStripeRedGreaterThan(img1, x, minR)
	startY2, _, err2 := FindLongestStripeRedGreaterThan(img2, x, minR)
	if err1 != nil || err2 != nil {
		return 0, fmt.Errorf("Ошибка поиска полоски: %v, %v", err1, err2)
	}
	return startY2 - startY1, nil
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
