package main

import (
	"bufio"
	"fmt"
	"log"
	"net/netip"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/spf13/cobra"
)

var supportedInputFormats = map[string]bool{
	strings.ToLower("clashRuleSet"):          true,
	strings.ToLower("clashRuleSetClassical"): true,
	strings.ToLower("maxmindMMDB"):           true,
	strings.ToLower("mihomoMRS"):             true,
	strings.ToLower("singboxSRS"):            true,
	strings.ToLower("surgeRuleSet"):          true,
	strings.ToLower("text"):                  true,
	strings.ToLower("v2rayGeoIPDat"):         true,
}

func init() {
	rootCmd.AddCommand(lookupCmd)

	lookupCmd.Flags().StringP("format", "f", "", "(Required) The input format. Available formats: text, v2rayGeoIPDat, maxmindMMDB, mihomoMRS, singboxSRS, clashRuleSet, clashRuleSetClassical, surgeRuleSet")
	lookupCmd.Flags().StringP("uri", "u", "", "URI of the input file, support both local file path and remote HTTP(S) URL. (Cannot be used with \"dir\" flag)")
	lookupCmd.Flags().StringP("dir", "d", "", "Path to the input directory. The filename without extension will be as the name of the list. (Cannot be used with \"uri\" flag)")
	lookupCmd.Flags().StringSliceP("searchlist", "l", []string{}, "The lists to search from, separated by comma")

	lookupCmd.MarkFlagRequired("format")
	lookupCmd.MarkFlagsOneRequired("uri", "dir")
	lookupCmd.MarkFlagsMutuallyExclusive("uri", "dir")
	lookupCmd.MarkFlagDirname("dir")
}

var lookupCmd = &cobra.Command{
	Use:     "lookup",
	Aliases: []string{"find"},
	Short:   "Lookup if specified IP or CIDR is in specified lists",
	Args:    cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		// Validate format
		format, _ := cmd.Flags().GetString("format")
		format = strings.ToLower(strings.TrimSpace(format))
		if _, found := supportedInputFormats[format]; !found {
			log.Fatal("unsupported input format")
		}

		// Set name
		name := "true"

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
			search := strings.ToLower(strings.TrimSpace(args[0]))
			if !isValidIPOrCIDR(search) {
				fmt.Println("false")
				return
			}

			execute(format, name, uri, dir, search, searchListStr)

		case false: // No search arg, run in REPL mode
			fmt.Println(`Enter IP or CIDR (type "exit" to quit):`)
			fmt.Print(">> ")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				search := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if search == "" {
					fmt.Println()
					fmt.Print(">> ")
					continue
				}
				if search == "exit" || search == `"exit"` {
					break
				}

				if !isValidIPOrCIDR(search) {
					fmt.Println("false")
					fmt.Println()
					fmt.Print(">> ")
					continue
				}

				execute(format, name, uri, dir, search, searchListStr)

				fmt.Println()
				fmt.Print(">> ")
			}
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
		}
	},
}

// Check if the input is a valid IP or CIDR
func isValidIPOrCIDR(search string) bool {
	if search == "" {
		return false
	}

	var err error
	switch strings.Contains(search, "/") {
	case true: // CIDR
		_, err = netip.ParsePrefix(search)
	case false: // IP
		_, err = netip.ParseAddr(search)
	}

	return err == nil
}

func execute(format, name, uri, dir, search, searchListStr string) {
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
