package image

import (
	"image"
)

// FindItemPositionsByTextColor находит центры цветных строк с названиями предметов.
// Алгоритм сначала находит все отдельные цветные строки, а затем группирует
// те, что находятся близко друг к другу, вычисляя для каждой группы единый центр.
func FindItemPositionsByTextColor(img image.Image, targetX int) []image.Point {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	type bar struct{ yStart, yEnd int }
	var allBars []bar

	// --- Этап 1: Находим все отдельные строки цветного текста ---
	const scanXStart = 70
	const minHorizontalPixels = 20
	const colorThreshold = 20

	inBar := false
	var barYStart int
	for y := 30; y < height; y++ {
		activePixelCount := 0
		for x := scanXStart; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			isGreen := g8 > r8+colorThreshold && g8 > b8+colorThreshold
			isRed := r8 > g8+colorThreshold && r8 > b8+colorThreshold
			if isGreen || isRed {
				activePixelCount++
			}
		}

		isRowActive := activePixelCount >= minHorizontalPixels
		if isRowActive && !inBar {
			inBar = true
			barYStart = y
		} else if !isRowActive && inBar {
			inBar = false
			allBars = append(allBars, bar{yStart: barYStart, yEnd: y - 1})
		}
	}
	if inBar {
		allBars = append(allBars, bar{yStart: barYStart, yEnd: height - 1})
	}

	// --- Этап 2: Группируем близкие строки и вычисляем центры ---
	var centers []image.Point
	if len(allBars) == 0 {
		return centers
	}

	const minDistanceY = 15 // Макс. расстояние для объединения в одну группу.
	currentGroup := allBars[0]

	for i := 1; i < len(allBars); i++ {
		nextBar := allBars[i]
		// Если следующая строка близко, она является частью текущей группы.
		if (nextBar.yStart - currentGroup.yEnd) < minDistanceY {
			// Расширяем границы группы.
			currentGroup.yEnd = nextBar.yEnd
		} else {
			// Следующая строка далеко - значит, предыдущая группа закончилась.
			// Вычисляем и сохраняем ее центр.
			centerY := currentGroup.yStart + (currentGroup.yEnd-currentGroup.yStart)/2
			centers = append(centers, image.Point{X: targetX, Y: centerY})
			// Начинаем новую группу.
			currentGroup = nextBar
		}
	}

	// Сохраняем центр последней группы.
	lastCenterY := currentGroup.yStart + (currentGroup.yEnd-currentGroup.yStart)/2
	centers = append(centers, image.Point{X: targetX, Y: lastCenterY})

	return centers
}
