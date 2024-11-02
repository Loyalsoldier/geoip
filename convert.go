package main

import (
	"log"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(convertCmd)
	convertCmd.PersistentFlags().StringP("config", "c", "config.json", "URI of the JSON format config file, support both local file path and remote HTTP(S) URL")
}

var convertCmd = &cobra.Command{
	Use:     "convert",
	Aliases: []string{"conv"},
	Short:   "Convert geoip data from one format to another by using config file",
	Run: func(cmd *cobra.Command, args []string) {
		configFile, _ := cmd.Flags().GetString("config")
		log.Println("Use config:", configFile)

		instance, err := lib.NewInstance()
		if err != nil {
			log.Fatal(err)
		}

		if err := instance.InitConfig(configFile); err != nil {
			log.Fatal(err)
		}

		if err := instance.Run(); err != nil {
			log.Fatal(err)
		}
	},
}
