package main

import (
	"fmt"
	"image/color"
	_ "image/png"
	"os"
	"strings"

	"github.com/disintegration/imaging"
)

var brailleMatrix = [4][2]int{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
}

func renderPNGToBraille(filename string, charWidth int, colored bool, invert bool) string {
	src, err := imaging.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", filename, err)
		return ""
	}

	bounds := src.Bounds()
	aspect := float64(bounds.Dy()) / float64(bounds.Dx())

	pixelWidth := charWidth * 2
	pixelHeight := int(float64(charWidth) * aspect * 2)
	pixelHeight = pixelHeight - (pixelHeight % 4)

	dst := imaging.Resize(src, pixelWidth, pixelHeight, imaging.Lanczos)

	var sb strings.Builder
	sb.Grow(charWidth * (pixelHeight / 4) * 30)

	for y := 0; y < pixelHeight; y += 4 {
		for x := 0; x < pixelWidth; x += 2 {

			var brailleVal int
			var rSum, gSum, bSum, count float64

			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					c := dst.At(x+dx, y+dy)
					_, _, _, a := c.RGBA()

					// Ignore fully transparent pixels
					if a == 0 {
						continue
					}

					lum := luminance(c)

					// INVERT LOGIC:
					// If invert is false: draw dots where the image is BRIGHT
					// If invert is true: draw dots where the image is DARK
					isDot := lum > 50.0
					if invert {
						isDot = lum < 200.0
					}

					if isDot {
						brailleVal += brailleMatrix[dy][dx]

						// ONLY grab colors for the pixels that actually form the dots
						// This stops the background from "muddying" the color
						r, g, b, _ := c.RGBA()
						rSum += float64(r >> 8)
						gSum += float64(g >> 8)
						bSum += float64(b >> 8)
						count++
					}
				}
			}

			brailleChar := string(rune(0x2800 + brailleVal))

			if colored && count > 0 {
				// Calculate the boosted color as a float FIRST
				rawR := (rSum / count) * 1.2
				rawG := (gSum / count) * 1.2
				rawB := (bSum / count) * 1.2

				// Cap it at 255 BEFORE converting to uint8 to prevent overflow
				if rawR > 255 {
					rawR = 255
				}
				if rawG > 255 {
					rawG = 255
				}
				if rawB > 255 {
					rawB = 255
				}

				avgR := uint8(rawR)
				avgG := uint8(rawG)
				avgB := uint8(rawB)

				fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm%s", avgR, avgG, avgB, brailleChar)
			} else if brailleVal > 0 {
				sb.WriteString(brailleChar)
			} else {
				sb.WriteString(" ")
			}
		}
		sb.WriteString("\x1b[0m\n")
	}

	return sb.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <image.png> [width] [colored] [invert]")
		fmt.Println("Example: go run main.go deadpool.png 80 true true")
		return
	}

	filename := os.Args[1]
	width := 80
	colored := true
	invert := false // Default to standard

	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &width)
	}
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%t", &colored)
	}
	if len(os.Args) > 4 {
		fmt.Sscanf(os.Args[4], "%t", &invert)
	}

	result := renderPNGToBraille(filename, width, colored, invert)
	fmt.Print(result)
}
