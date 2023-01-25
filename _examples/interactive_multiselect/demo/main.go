package main

import (
	"fmt"

	"github.com/solo-io/pterm"
)

func main() {
	var options []string

	for i := 0; i < 100; i++ {
		options = append(options, fmt.Sprintf("Option %d", i))
	}

	for i := 0; i < 5; i++ {
		options = append(options, fmt.Sprintf("You can use fuzzy searching (%d)", i))
	}

	selectedOptions, _ := pterm.DefaultInteractiveMultiselect.WithOptions(options).Show()
	pterm.Info.Printfln("Selected options: %s", pterm.Green(selectedOptions))
}
