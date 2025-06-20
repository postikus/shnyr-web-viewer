package scpript1

import (
	"fmt"
	// "image/png"
	//"github.com/otiai10/gosseract/v2"
	"github.com/tarm/serial"
	//_ "gocv.io/x/gocv"
	"image"
	//"image/png"
	"io/ioutil"
	//"image/color"
	//"image/draw"
	"octopus/internal/config"
	imageInternal "octopus/internal/image"
	"octopus/internal/screenshot"
	"octopus/internal/scripts"
)

var Run = func(port *serial.Port, c *config.Config) {
	var saveItmOffersScreenShotBasedOnPixel = func() {
		img, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemOffersListButtons)
		r, _, _, _ := imageInternal.GetPixelColor(img, 2, 2)
		if r > 26 {
			screenshot.SaveItemOffersWithButtondScreenshot(c)
		} else {
			screenshot.SaveItemOffersWithoutButtondScreenshot(c)
		}
	}

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
			saveItmOffersScreenShotBasedOnPixel()
		}

		scripts.ClickCoordinates(port, c, c.Click.Back)
	}

	var clickEveryItemAnsScreenShot = func(img image.Image) {
		//прокликиваем первую страницу
		columns, _ := imageInternal.FindNonBlackPixelCoordinatesInColumn(img, 35)
		if len(columns) > 2 {
			var ys = imageInternal.FindSegments(columns, 10)

			for _, y := range ys {
				clickItem(config.Coordinates{Y: y + c.Screenshot.ItemList.Y, X: 50 + c.Screenshot.ItemList.X})
			}
		}

		//clickItem(c.Click.Item2)
	}

	// //берем в фокус и делаем скрин
	// scripts.ClickCoordinates(port, c, c.Click.Item1)
	// img, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemList)
	// clickEveryItemAnsScreenShot(img)

	// берем в фокус и делаем скрин
	scripts.ClickCoordinates(port, c, c.Click.Item1)

	cycles := 0
	for cycles < 50 {
		img, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemList)
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, c.Click.Button2)
		img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, c.Click.Button3)
		img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, c.Click.Button4)
		img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, c.Click.Button5)
		img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
		clickEveryItemAnsScreenShot(img)

		scripts.ClickCoordinates(port, c, c.Click.Button6)
		img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
		clickEveryItemAnsScreenShot(img)

		ButtonsImg, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemOffersListButtons)
		SixButtonPx, _, _, _ := imageInternal.GetPixelColor(ButtonsImg, 124, 14)
		maxSixButtonClicks := 0

		for SixButtonPx > 30 && maxSixButtonClicks < 50 {
			scripts.ClickCoordinates(port, c, c.Click.Button6)
			img, _ = screenshot.CaptureScreenshot(c.Screenshot.ItemList)
			clickEveryItemAnsScreenShot(img)
			ButtonsImg, _ := screenshot.CaptureScreenshot(c.Screenshot.ItemOffersListButtons)
			SixButtonPx, _, _, _ = imageInternal.GetPixelColor(ButtonsImg, 124, 14)
			maxSixButtonClicks += 1
		}

		scripts.ClickCoordinates(port, c, c.Click.Back)
		scripts.ClickCoordinates(port, c, config.Coordinates{X: 35, Y: 107})

		cycles += 1
	}

}
