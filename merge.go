package main

import (
	"log"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/special"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.PersistentFlags().StringP("onlyiptype", "t", "", "The only IP type to output, available options: \"ipv4\", \"ipv6\"")
}

var mergeCmd = &cobra.Command{
	Use:     "merge",
	Aliases: []string{"m"},
	Short:   "Merge plaintext IP & CIDR from standard input, then print to standard output",
	Run: func(cmd *cobra.Command, args []string) {
		otype, _ := cmd.Flags().GetString("onlyiptype")
		otype = strings.ToLower(strings.TrimSpace(otype))

		if otype != "" && otype != "ipv4" && otype != "ipv6" {
			log.Fatal("invalid argument onlyiptype: ", otype)
		}

		instance, err := lib.NewInstance()
		if err != nil {
			log.Fatal(err)
		}

		instance.AddInput(getInputForMerge())
		instance.AddOutput(getOutputForMerge(otype))

		if err := instance.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func getInputForMerge() lib.InputConverter {
	return &special.Stdin{
		Type:        special.TypeStdin,
		Action:      lib.ActionAdd,
		Description: special.DescStdin,
		Name:        "temp",
	}
}

func getOutputForMerge(otype string) lib.OutputConverter {
	switch lib.IPType(otype) {
	case lib.IPv4:
		return &special.Stdout{
			Type:        special.TypeStdout,
			Action:      lib.ActionOutput,
			Description: special.DescStdout,
			OnlyIPType:  lib.IPv4,
		}

	case lib.IPv6:
		return &special.Stdout{
			Type:        special.TypeStdout,
			Action:      lib.ActionOutput,
			Description: special.DescStdout,
			OnlyIPType:  lib.IPv6,
		}

	default:
		return &special.Stdout{
			Type:        special.TypeStdout,
			Action:      lib.ActionOutput,
			Description: special.DescStdout,
		}
	}

}
