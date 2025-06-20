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
	marginY := gameWindow.Y + topOffset + 45

	fmt.Printf("marginX, marginY: %v %v\n", marginX, marginY)

	var checkAndScreenScroll = func(counter int, x int) (int, int) {
		img, _ := screenshot.CaptureScreenshot(config.CoordinatesWithSize{X: c.Screenshot.ItemOffersListWithoutButtons.X, Y: c.Screenshot.ItemOffersListWithoutButtons.Y, Width: 260, Height: c.Screenshot.ItemOffersListWithoutButtons.Height})
		r, _, _, _ := imageInternal.GetPixelColor(img, 255, 290)
		if r < 50 {
			scripts.ScrollDown(port, c, x)
		}
		return counter + 1, r
	}

	var checkAndClickScreenScroll = func(counter int) (int, int) {
		img, _ := screenshot.CaptureScreenshot(config.CoordinatesWithSize{X: c.Screenshot.ItemOffersListWithoutButtons.X, Y: c.Screenshot.ItemOffersListWithoutButtons.Y, Width: 260, Height: c.Screenshot.ItemOffersListWithoutButtons.Height})
		r, _, _, _ := imageInternal.GetPixelColor(img, 255, 315)
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

	var makeScreenShotsWithScroll = func() bool {
		counter := 0
		maxCounter := 100
		scrollRPx := 26

		// Список для хранения всех скриншотов
		var screenshots []image.Image
		var smallScreenshots []image.Image

		img, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemOffersListWithoutButtons)
		screenshots = append(screenshots, img)

		img, _ = screenshot.CaptureScreenshot(config.CoordinatesWithSize{X: c.Screenshot.ItemOffersListWithoutButtons.X, Y: c.Screenshot.ItemOffersListWithoutButtons.Y, Width: 260, Height: c.Screenshot.ItemOffersListWithoutButtons.Height})
		scrollRPx, _, _, _ = imageInternal.GetPixelColor(img, 242, 2)

		if scrollRPx > 26 {
			scrollRPx = 26
			for counter < maxCounter && scrollRPx < 50 {
				counter, scrollRPx = checkAndScreenScroll(counter, 1)
				if scrollRPx < 50 {
					img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemOffersListWithoutButtons)
					screenshots = append(screenshots, img)
				}
			}

			scrollRPx = 26
			scripts.ClickCoordinates(port, c, c.Click.Scroll)
			for counter < maxCounter && scrollRPx < 50 {
				counter, scrollRPx = checkAndClickScreenScroll(counter)
				if scrollRPx < 50 {
					img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemOffersListWithoutButtons)
					smallScreenshots = append(smallScreenshots, img)
				}
			}

			finalImage, _ := imageInternal.CombineImages(screenshots, smallScreenshots)
			combinedImg := imageInternal.CropOpacityPixel(finalImage)

			fileCount, err := countFilesInDir("./imgs")
			fileName := fmt.Sprintf("%s/screenshot_combined_%d.png", "./imgs", fileCount)
			err = imageInternal.SaveCombinedImage(combinedImg, fileName)
			if err != nil {
				return false
			}
			scripts.ScrollUp(port, c, counter)
			return true
		}
		return false
	}

	var clickItem = func(item config.Coordinates) {
		scripts.ClickCoordinates(port, c, item)
		combinedSaved := makeScreenShotsWithScroll()
		if !combinedSaved {
			screenshot.SaveScreenshot(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 364})
		}

		scripts.ClickCoordinates(port, c, config.Coordinates{X: marginX + c.Click.Back.X, Y: marginY + c.Click.Back.Y})
	}

	var clickEveryItemAnsScreenShot = func(img image.Image) {
		// прокликиваем первую страницу
		columns, _ := imageInternal.FindNonBlackPixelCoordinatesInColumn(img, 35)
		if len(columns) > 2 {
			var ys = imageInternal.FindSegments(columns, 5)

			fmt.Printf("ys: %v\n", ys)

			for _, y := range ys {
				clickItem(config.Coordinates{Y: y + marginY + c.Screenshot.ItemList.Y, X: marginX + c.Screenshot.ItemList.X})
			}
		}

		// clickItem(config.Coordinates{X: marginX + c.Click.Item1.X, Y: marginY + c.Click.Item1.Y})
	}

	//берем в фокус и делаем скрин
	scripts.ClickCoordinates(port, c, c.Click.Item1)
	img, _ = screenshot.SaveScreenshot(config.CoordinatesWithSize{X: marginX, Y: marginY, Width: 300, Height: 364})
	clickEveryItemAnsScreenShot(img)

	// // берем в фокус и делаем скрин
	// scripts.ClickCoordinates(port, c, c.Click.Item1)

	// cycles := 0
	// for cycles < 50 {
	// 	img, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemList)
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, c.Click.Button2)
	// 	img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, c.Click.Button3)
	// 	img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, c.Click.Button4)
	// 	img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, c.Click.Button5)
	// 	img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
	// 	clickEveryItemAnsScreenShot(img)

	// 	scripts.ClickCoordinates(port, c, c.Click.Button6)
	// 	img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
	// 	clickEveryItemAnsScreenShot(img)

	// 	ButtonsImg, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemOffersListButtons)
	// 	SixButtonPx, _, _, _ := imageInternal.GetPixelColor(ButtonsImg, 124, 14)
	// 	maxSixButtonClicks := 0

	// 	for SixButtonPx > 30 && maxSixButtonClicks < 50 {
	// 		scripts.ClickCoordinates(port, c, c.Click.Button6)
	// 		img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
	// 		clickEveryItemAnsScreenShot(img)
	// 		ButtonsImg, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemOffersListButtons)
	// 		SixButtonPx, _, _, _ = imageInternal.GetPixelColor(ButtonsImg, 124, 14)
	// 		maxSixButtonClicks += 1
	// 	}

	// 	scripts.ClickCoordinates(port, c, c.Click.Back)
	// 	scripts.ClickCoordinates(port, c, config.Coordinates{X: 35, Y: 107})

	// 	cycles += 1
	// }

}
