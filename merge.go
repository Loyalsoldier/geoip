package main

import (
	"log"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/spf13/cobra"
)

const tempConfig = `
{
  "input": [
    {
      "type": "stdin",
      "action": "add",
      "args": {
        "name": "temp"
      }
    }
  ],
  "output": [
    {
      "type": "stdout",
      "action": "output"
    }
  ]
}
`

const tempConfigWithIPv4 = `
{
  "input": [
    {
      "type": "stdin",
      "action": "add",
      "args": {
        "name": "temp"
      }
    }
  ],
  "output": [
    {
      "type": "stdout",
      "action": "output",
      "args": {
        "onlyIPType": "ipv4"
      }
    }
  ]
}
`

const tempConfigWithIPv6 = `
{
  "input": [
    {
      "type": "stdin",
      "action": "add",
      "args": {
        "name": "temp"
      }
    }
  ],
  "output": [
    {
      "type": "stdout",
      "action": "output",
      "args": {
        "onlyIPType": "ipv6"
      }
    }
  ]
}
`

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

		var configBytes []byte
		switch lib.IPType(otype) {
		case lib.IPv4:
			configBytes = []byte(tempConfigWithIPv4)
		case lib.IPv6:
			configBytes = []byte(tempConfigWithIPv6)
		default:
			configBytes = []byte(tempConfig)
		}

		instance, err := lib.NewInstance()
		if err != nil {
			log.Fatal(err)
		}

		if err := instance.InitFromBytes(configBytes); err != nil {
			log.Fatal(err)
		}

		if err := instance.Run(); err != nil {
			log.Fatal(err)
		}
	},
}
