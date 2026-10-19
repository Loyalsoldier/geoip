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
	input, err := special.NewStdin(
		lib.ActionAdd,
		special.WithStdinName("temp"),
	)
	if err != nil {
		log.Fatal(err)
	}

	return input
}

func getOutputForMerge(otype string) lib.OutputConverter {
	var output lib.OutputConverter
	var err error

	switch lib.IPType(otype) {
	case lib.IPv4:
		output, err = special.NewStdout(
			lib.ActionOutput,
			special.WithStdoutOnlyIPType(lib.IPv4),
		)

	case lib.IPv6:
		output, err = special.NewStdout(
			lib.ActionOutput,
			special.WithStdoutOnlyIPType(lib.IPv6),
		)

	default:
		output, err = special.NewStdout(lib.ActionOutput)
	}

	if err != nil {
		log.Fatal(err)
	}

	return output
}
