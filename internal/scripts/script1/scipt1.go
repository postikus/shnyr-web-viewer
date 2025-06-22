package scpript1

import (
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"octopus/internal/config"
	imageInternal "octopus/internal/image"
	"octopus/internal/screen"
	"octopus/internal/screenshot"
	"octopus/internal/scripts"

	"github.com/tarm/serial"
)

var Run = func(port *serial.Port, c *config.Config) {

	// Количество пикселей для отрезания сверху
	topOffset := 23

	// Делаем скриншот всего экрана
	img, err := screen.CaptureFullScreen()
	if err != nil {
		log.Fatalf("Ошибка захвата экрана: %v", err)
	}

	// Обрезаем верхние пиксели
	bounds := img.Bounds()
	croppedImg := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()-topOffset))
	for y := topOffset; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			croppedImg.Set(x, y-topOffset, img.At(x, y))
		}
	}

	// Ищем окно
	gameWindow, err := imageInternal.FindGameWindow(croppedImg)
	if err != nil {
		fmt.Printf("Окно не найдено: %v\n", err)
		return
	}

	marginX := gameWindow.X - 150
	marginY := gameWindow.Y + topOffset + 48

	fmt.Printf("marginX, marginY: %v %v\n", marginX, marginY)

	var captureScreenShot = func() image.Image {
		img, _ := screenshot.CaptureScreenshot(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361})
		return img
	}

	var saveScreenShot = func() image.Image {
		img, _ := screenshot.SaveScreenshot(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 361})
		return img
	}

	var checkAndScreenScroll = func(counter int, x int) (int, int) {
		img := captureScreenShot()
		r, _, _, _ := imageInternal.GetPixelColor(img, 283, 315)
		fmt.Printf("r: %v\n", r)
		if r < 50 {
			scripts.ScrollDown(port, c, x)
		}
		return counter + 1, r
	}

	var checkAndClickScreenScroll = func(counter int) (int, int) {
		img := captureScreenShot()
		r, _, _, _ := imageInternal.GetPixelColor(img, 283, 332)
		if r < 50 {
			scripts.FastClick(port, c)
		}
		return counter + 1, r
	}

	countFilesInDir := func(dir string) (int, error) {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return 0, fmt.Errorf("не удалось прочитать папку: %v", err)
		}

		// Возвращаем количество файлов в папке
		return len(files), nil
	}

	var captureScreenShotsWithScroll = func() bool {
		counter := 0
		maxCounter := 20
		scrollRPx := 26

		// Список для хранения всех скриншотов
		var screenshots []image.Image
		var smallScreenshots []image.Image

		img := captureScreenShot()
		screenshots = append(screenshots, img)
		scrollRPx, scrollGPx, scrollBPx, _ := imageInternal.GetPixelColor(img, 290, 15)
		fmt.Printf("scrollRPx: %v %v %v\n", scrollRPx, scrollGPx, scrollBPx)

		if scrollRPx > 26 {
			scrollRPx = 26
			for counter < maxCounter && scrollRPx < 50 {
				counter, scrollRPx = checkAndScreenScroll(counter, 1)
				if scrollRPx < 50 {
					img = captureScreenShot()
					screenshots = append(screenshots, img)
				}
			}

			scrollRPx = 26
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Scroll.X, Y: marginY + c.Click.Scroll.Y})
			for counter < maxCounter && scrollRPx < 50 {
				counter, scrollRPx = checkAndClickScreenScroll(counter)
				if scrollRPx < 50 {
					img = captureScreenShot()
					smallScreenshots = append(smallScreenshots, img)
				}
			}

			finalImage, _ := imageInternal.CombineImages(screenshots, smallScreenshots)
			combinedImg := imageInternal.CropOpacityPixel(finalImage)

			fileCount, _ := countFilesInDir("./imgs")
			fileName := fmt.Sprintf("%s/screenshot_combined_%d.png", "./imgs", fileCount)
			err = imageInternal.SaveCombinedImage(combinedImg, fileName)
			if err != nil {
				return false
			}
			scripts.ScrollUp(port, c, counter+5)
			return true
		}
		return false
	}

	var clickItem = func(item config.Coordinates) {
		scripts.ClickCoordinates(port, c, item)
		combinedSaved := captureScreenShotsWithScroll()
		if !combinedSaved {
			saveScreenShot()
		}

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
	}

	var clickEveryItemAnsScreenShot = func(img image.Image) {
		// прокликиваем первую страницу
		points := imageInternal.FindItemPositionsByTextColor(img, 100)
		if len(points) > 2 {
			for _, point := range points {
				clickItem(config.Coordinates{Y: point.Y + marginY, X: marginX + point.X})
			}
		}

		// clickItem(config.Coordinates{X: marginX + c.Click.Item8.X, Y: marginY + c.Click.Item8.Y})
	}

	//берем в фокус и делаем скрин
	// scripts.ClickCoordinates(port, c, c.Click.Item1)
	// img = saveScreenShot()
	// clickEveryItemAnsScreenShot(img)

	// берем в фокус
	scripts.ClickCoordinates(port, c, c.Click.Item1)

	cycles := 0
	for cycles < 1 {
		img := captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button2.X, Y: marginY + c.Click.Button2.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button3.X, Y: marginY + c.Click.Button3.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button4.X, Y: marginY + c.Click.Button4.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button5.X, Y: marginY + c.Click.Button5.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
		img = captureScreenShot()
		clickEveryItemAnsScreenShot(img)

		img = captureScreenShot()
		SixButtonPx, _, _, _ := imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
		maxSixButtonClicks := 0

		for SixButtonPx > 30 && maxSixButtonClicks < 50 {
			scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Button6.X, Y: marginY + c.Click.Button6.Y})
			img = captureScreenShot()
			clickEveryItemAnsScreenShot(img)
			img = captureScreenShot()
			SixButtonPx, _, _, _ = imageInternal.GetPixelColor(img, c.Click.Button6.X, 35)
			maxSixButtonClicks += 1
		}

		// scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
		// scripts.ClickCoordinates(port, c, config.Coordinates{X: 35, Y: 107})

		cycles += 1
	}

}
