package image

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"

	"github.com/nfnt/resize"
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

func FindNonBlackPixelCoordinatesInColumn(img image.Image, x int) ([]int, error) {
	// Массив для хранения координат пикселей, отличных от черного
	var nonBlackPixelCoordinates []int

	// Получаем высоту изображения
	height := img.Bounds().Max.Y

	// Проходим по всем пикселям в столбце
	for y := 0; y < height; y++ {
		// Получаем цвет пикселя

		// Проверяем, что пиксель не черный
		r, g, b, _ := GetPixelColor(img, x, y)

		if (g > r+50 || g > b+50) || r > g+50 || r > b+50 {
			// Если пиксель не черный, добавляем его координату Y в список
			nonBlackPixelCoordinates = append(nonBlackPixelCoordinates, y)
		}
	}

	// Возвращаем координаты пикселей, которые не черные
	return nonBlackPixelCoordinates, nil
}

func FindSegments(arr []int, threshold int) []int {
	var segments []int
	start := 0

	// Проходим по всему массиву
	for i := 1; i < len(arr); i++ {
		// Если разница между соседними элементами меньше threshold
		if arr[i]-arr[i-1] < threshold {
			continue
		} else {
			// Если разница больше или равна threshold, то находим центр текущего отрезка
			// Центр отрезка - это среднее значение первого и последнего элемента отрезка
			segmentCenter := (arr[start] + arr[i-1]) / 2
			segments = append(segments, segmentCenter)
			start = i
		}
	}

	// Для последнего отрезка
	segmentCenter := (arr[start] + arr[len(arr)-1]) / 2
	segments = append(segments, segmentCenter)

	return segments
}

func ChangeTopAndBottomRowToColor(img image.Image, newColor color.RGBA) image.Image {
	// Преобразуем изображение в формат RGBA, если оно не в этом формате
	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		// Если это не *image.RGBA, создаем новый RGBA и копируем данные
		rgbaImg = image.NewRGBA(img.Bounds())
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				rgbaImg.Set(x, y, img.At(x, y))
			}
		}
	}

	// Меняем все пиксели на координате y = 1
	width := img.Bounds().Dx() // Ширина изображения
	y := 1                     // Координата строки, которую нужно изменить
	for x := 0; x < width; x++ {
		rgbaImg.Set(x, y, newColor) // Изменяем пиксель по координате (x, y)
	}
	y = img.Bounds().Dy()
	for x := 0; x < width; x++ {
		rgbaImg.Set(x, y, newColor) // Изменяем пиксель по координате (x, y)
	}

	return rgbaImg
}

var GetScrollHeightDiff = func(img1 image.Image, img2 image.Image) int {
	pixel1, _ := FindFirstNonBlackPixel(img1, 248)
	pixel2, _ := FindFirstNonBlackPixel(img2, 248)
	fmt.Println(pixel1, pixel2)
	diff := pixel1 - pixel2
	return diff
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

func FindFirstNonBlackPixel(img image.Image, x int) (int, error) {
	// Получаем размеры изображения
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Проверяем, что x находится в пределах изображения
	if x < 0 || x >= width {
		return -1, fmt.Errorf("координата x выходит за пределы изображения")
	}

	// Проходим по всем пикселям в столбце по оси y от 0 до высоты изображения
	for y := 0; y < height; y++ {
		// Получаем пиксель в координатах (x, y)
		r, g, b, _ := GetPixelColor(img, x, y)
		// Проверяем, является ли пиксель черным

		if r > 50 || g > 50 || b > 50 {
			// Если пиксель не черный, возвращаем его координату по y
			return y, nil
		}
	}

	// Если не найдено ни одного пикселя, отличного от черного
	return -1, nil
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

func PreprocessImage(img image.Image) image.Image {
	// Получаем размеры изображения
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	// Создаем новое изображение с теми же размерами
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))

	// Проходим по каждому пикселю изображения
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Получаем цвет пикселя
			originalColor := img.At(x, y)
			r, g, b, _ := originalColor.RGBA()

			// Преобразуем цвет в 8-битные значения
			r = r >> 8
			g = g >> 8
			b = b >> 8
			// Если пиксель зеленый, меняем его на белый
			if g > 120 {
				newImg.Set(x, y, color.White)
			} else if r > 120 {
				// Если пиксель красный, оставляем его без изменений
				newImg.Set(x, y, color.White)
			} else if r > 80 && g > 80 && b > 80 {
				// Если пиксель белый, оставляем его без изменений
				newImg.Set(x, y, color.White)
			} else {
				// Все остальные пиксели меняем на черные
				newImg.Set(x, y, color.Black)
				//newImg.Set(x, y, color.RGBA{17, 17, 17, 255})
			}
		}
	}

	return newImg
}

func ScaleImage(img image.Image) image.Image {
	// Получаем размеры исходного изображения
	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y

	// Увеличиваем размеры изображения в 7 раз
	newImage := resize.Resize(uint(width*7), uint(height*7), img, resize.Lanczos3)

	return newImage
}

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

func OCR(imagePath string) error {
	// Запускаем OCR для извлечения текста
	cmd := exec.Command("go", "run", "./cmd/ocr_runner/main.go", imagePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка при выполнении OCR: %v, вывод: %s", err, string(output))
	}

	fmt.Printf("OCR результат для %s:\n%s\n", imagePath, string(output))
	return nil
}
