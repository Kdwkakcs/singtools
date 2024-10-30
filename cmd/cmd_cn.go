package main

import (
	"github.com/Dkwkoaca/singtools/cn"
	"github.com/spf13/cobra"
)

var cnCommand = &cobra.Command{
	Use:   "cn",
	Short: "Get cn node from config file, which containing the links",
	Long:  `Get cn node from config file, which containing the links`,
	Run:   cnNodeTest,
}

var (
	inputFile  string
	urlPath    string
	dbPath     string
	outputFile string
	outputJson string
)

func init() {
	cnCommand.Flags().StringVarP(&inputFile, "input", "i", "", "input config file")
	cnCommand.Flags().StringVarP(&urlPath, "url", "u", "", "url file path")
	cnCommand.Flags().StringVarP(&dbPath, "db", "d", "", "db file path")
	cnCommand.Flags().StringVarP(&outputFile, "output", "o", "", "output file path")
	cnCommand.Flags().StringVarP(&outputJson, "json", "j", "", "json file to save the links")

	mainCommand.AddCommand(cnCommand)
}

func cnNodeTest(cmd *cobra.Command, args []string) {
	if inputFile == "" && urlPath == "" {
		cmd.Help()
	}
	cn.CNNodeTest(inputFile, urlPath, dbPath, outputFile, outputJson)
}
