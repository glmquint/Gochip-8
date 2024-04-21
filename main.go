package main

import (
	Chip "gochip/chip"
	"time"
)

func display(chip *Chip.Chip) {
	for {
		chip.Draw()
		time.Sleep(time.Second / 60)
	}
}

func main() {
	chip := Chip.NewChip("roms/octojam4title.ch8")
	// go display(chip)
	chip.Run([]uint16{})
}
