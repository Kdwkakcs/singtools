package main

import (

	// "github.com/spyzhov/ajson"

	"github.com/spf13/cobra"
)

var mainCommand = &cobra.Command{
	Use:   "lite",
	Short: "Lite speed test on all protocols supported by sing-box ",
	Long:  `Lite speed test on all protocols supported by sing-box, using only for me`,
}
