package main

import (
	"github.com/Dkwkoaca/singtools/get"
	"github.com/spf13/cobra"
)

var getCommand = &cobra.Command{
	Use:   "get",
	Short: "Get node from config file, which containing the links",
	Long:  `Get node from config file, which containing the links`,
	Run:   getLinks,
}

var defaultConfig get.GetConfig

func init() {
	getCommand.Flags().StringVarP(&defaultConfig.InputFile, "input", "i", "", "input config file")
	getCommand.Flags().StringVarP(&defaultConfig.OutputFile, "output", "o", "", "output file path")
	getCommand.Flags().StringVarP(&defaultConfig.Category, "category", "t", "tg", "output file path")
	getCommand.Flags().StringVarP(&defaultConfig.SaveFile, "save", "s", "", "file to save the links")

	mainCommand.AddCommand(getCommand)
}

func getLinks(cmd *cobra.Command, args []string) {
	if defaultConfig.InputFile == "" || defaultConfig.OutputFile == "" {
		cmd.Help()
	}
	get.GetProxies(defaultConfig)
}
