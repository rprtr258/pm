// Generates test data for log viewers containing colors and fonts.
package main

import (
	"fmt"

	"github.com/deref/rgbterm"
)

func colors8() {
	fmt.Println("8 Colors")
	for i := 0; i < 8; i++ {
		fmt.Printf("\u001b[%dmX", 30+i) //nolint:gomnd // color offset
	}
	fmt.Println("\u001b[0m")
}

func colors16() {
	fmt.Println("16 Colors")
	for i := 0; i < 16; i++ {
		fmt.Printf("\u001b[%d;1mX", 30+i) //nolint:gomnd // color offset
	}
	fmt.Println("\u001b[0m")
}

func colors256() {
	fmt.Println("256 Colors")
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			fmt.Printf("\u001b[38;5;%04dmX", i*16+j)
		}
		fmt.Println("\u001b[0m")
	}
}

func colors24bit() {
	fmt.Println("24-bit Color")
	i := 0
	for hue := 0; hue < 256; hue++ {
		r, g, b := rgbterm.HSLtoRGB(float64(hue)/256.0, 0.7, 0.5) //nolint:gomnd // iterate hue
		fmt.Print(rgbterm.FgString("X", r, g, b))
		i++
		if i%32 == 0 {
			fmt.Println()
		}
	}
}

func decorations() {
	fmt.Println("decorations")
	fmt.Println("\u001b[1mBold\u001b[0m")
	fmt.Println("\u001b[2mFaint\u001b[0m")
	fmt.Println("\u001b[3mItalic\u001b[0m")
	fmt.Println("\u001b[4mUnderline\u001b[0m")
	fmt.Println("\u001b[5mSlow Blink\u001b[0m")
	fmt.Println("\u001b[6mFast Blink\u001b[0m") // Unsupported in Terminal.app on Mac.
	fmt.Println("\u001b[7mInvert\u001b[0m")
	fmt.Println("\u001b[8mConceal\u001b[0m")
	fmt.Println("\u001b[9mStrikethrough\u001b[0m") // Unsupported in Terminal.app on Mac.
}

func main() {
	colors8()
	fmt.Println()
	colors16()
	fmt.Println()
	colors256()
	fmt.Println()
	colors24bit()
	fmt.Println()
	decorations()
}
