package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/spf13/cobra"
)

var supportedInputFormats = map[string]bool{
	strings.ToLower("maxmindMMDB"):           true,
	strings.ToLower("clashRuleSetClassical"): true,
	strings.ToLower("clashRuleSet"):          true,
	strings.ToLower("surgeRuleSet"):          true,
	strings.ToLower("text"):                  true,
	strings.ToLower("singboxSRS"):            true,
	strings.ToLower("v2rayGeoIPDat"):         true,
}

func init() {
	rootCmd.AddCommand(lookupCmd)

	lookupCmd.Flags().StringP("format", "f", "", "The input format, available options: text, v2rayGeoIPDat, maxmindMMDB, singboxSRS, clashRuleSet, clashRuleSetClassical, surgeRuleSet")
	lookupCmd.Flags().StringP("name", "n", "", "The name of the list, use with \"uri\" flag")
	lookupCmd.Flags().StringP("uri", "u", "", "URI of the input file, support both local file path and remote HTTP(S) URL")
	lookupCmd.Flags().StringP("dir", "d", "", "Path to the input directory. The filename without extension will be as the name of the list")
	lookupCmd.Flags().StringSliceP("searchlist", "l", []string{}, "The lists to search from, separated by comma")

	lookupCmd.MarkFlagRequired("format")
	lookupCmd.MarkFlagsOneRequired("uri", "dir")
	lookupCmd.MarkFlagsRequiredTogether("name", "uri")
	lookupCmd.MarkFlagDirname("dir")
}

var lookupCmd = &cobra.Command{
	Use:     "lookup",
	Aliases: []string{"find"},
	Short:   "Lookup specified IP or CIDR in specified lists",
	Args:    cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		// Validate format
		format, _ := cmd.Flags().GetString("format")
		format = strings.ToLower(strings.TrimSpace(format))
		if _, found := supportedInputFormats[format]; !found {
			log.Fatal("unsupported input format")
		}

		// Get name
		name, _ := cmd.Flags().GetString("name")
		name = strings.ToLower(strings.TrimSpace(name))

		// Get uri
		uri, _ := cmd.Flags().GetString("uri")

		// Get dir
		dir, _ := cmd.Flags().GetString("dir")

		// Get searchlist
		searchList, _ := cmd.Flags().GetStringSlice("searchlist")
		searchListStr := strings.Join(searchList, `", "`)
		if searchListStr != "" {
			searchListStr = fmt.Sprint(`"`, searchListStr, `"`) // `"cn", "en"`
		}

		switch len(args) > 0 {
		case true: // With search arg, run in once mode
			search := strings.ToLower(args[0])
			config := generateConfigForLookup(format, name, uri, dir, search, searchListStr)

			instance, err := lib.NewInstance()
			if err != nil {
				log.Fatal(err)
			}
			if err := instance.InitFromBytes([]byte(config)); err != nil {
				log.Fatal(err)
			}

			if err := instance.Run(); err != nil {
				log.Fatal(err)
			}

		case false: // No search arg, run in REPL mode
			fmt.Println("Enter IP or CIDR (type `exit` to quit):")
			fmt.Print(">> ")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				search := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if search == "exit" {
					break
				}
				config := generateConfigForLookup(format, name, uri, dir, search, searchListStr)

				instance, err := lib.NewInstance()
				if err != nil {
					log.Fatal(err)
				}
				if err := instance.InitFromBytes([]byte(config)); err != nil {
					log.Fatal(err)
				}
				if err := instance.Run(); err != nil {
					log.Fatal(err)
				}
				fmt.Println()
				fmt.Print(">> ")
			}
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
		}
	},
}

func generateConfigForLookup(format, name, uri, dir, search, searchListStr string) string {
	return fmt.Sprintf(`
{
	"input": [
		{
			"type": "%s",
			"action": "add",
			"args": {
				"name": "%s",
				"uri": "%s",
				"inputDir": "%s"
			}
		}
	],
	"output": [
		{
			"type": "lookup",
			"action": "output",
			"args": {
				"search": "%s",
				"searchList": [%s]
			}
		}
	]
}
`, format, name, uri, dir, search, searchListStr)
}
