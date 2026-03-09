package main

import (
	"bufio"
	"fmt"
	"log"
	"net/netip"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/maxmind"
	"github.com/Loyalsoldier/geoip/plugin/mihomo"
	"github.com/Loyalsoldier/geoip/plugin/plaintext"
	"github.com/Loyalsoldier/geoip/plugin/singbox"
	"github.com/Loyalsoldier/geoip/plugin/special"
	"github.com/Loyalsoldier/geoip/plugin/v2ray"
	"github.com/spf13/cobra"
)

var supportedInputFormats = map[string]bool{
	strings.ToLower("clashRuleSet"):          true,
	strings.ToLower("clashRuleSetClassical"): true,
	strings.ToLower("dbipCountryMMDB"):       true,
	strings.ToLower("ipinfoCountryMMDB"):     true,
	strings.ToLower("maxmindMMDB"):           true,
	strings.ToLower("mihomoMRS"):             true,
	strings.ToLower("singboxSRS"):            true,
	strings.ToLower("surgeRuleSet"):          true,
	strings.ToLower("text"):                  true,
	strings.ToLower("v2rayGeoIPDat"):         true,
}

func init() {
	rootCmd.AddCommand(lookupCmd)

	lookupCmd.Flags().StringP("format", "f", "", "(Required) The input format. Available formats: text, v2rayGeoIPDat, maxmindMMDB, dbipCountryMMDB, ipinfoCountryMMDB, mihomoMRS, singboxSRS, clashRuleSet, clashRuleSetClassical, surgeRuleSet")
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

		switch len(args) > 0 {
		case true: // With search arg, run in once mode
			search := strings.ToLower(strings.TrimSpace(args[0]))
			if !isValidIPOrCIDR(search) {
				fmt.Println("false")
				return
			}

			instance, err := lib.NewInstance()
			if err != nil {
				log.Fatal(err)
			}

			instance.AddInput(getInputForLookup(format, name, uri, dir))
			instance.AddOutput(getOutputForLookup(search, searchList...))

			if err := instance.Run(); err != nil {
				log.Fatal(err)
			}

		case false: // No search arg, run in REPL mode
			instance, err := lib.NewInstance()
			if err != nil {
				log.Fatal(err)
			}
			instance.AddInput(getInputForLookup(format, name, uri, dir))

			container := lib.NewContainer()
			if err := instance.RunInput(container); err != nil {
				log.Fatal(err)
			}

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

				instance.ResetOutput()
				instance.AddOutput(getOutputForLookup(search, searchList...))

				if err := instance.RunOutput(container); err != nil {
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

func getInputForLookup(format, name, uri, dir string) lib.InputConverter {
	var input lib.InputConverter

	switch strings.ToLower(format) {
	case strings.ToLower(maxmind.TypeGeoLite2CountryMMDBIn):
		input = maxmind.NewGeoLite2CountryMMDBIn(
			maxmind.TypeGeoLite2CountryMMDBIn,
			maxmind.DescGeoLite2CountryMMDBIn,
			lib.ActionAdd,
			maxmind.WithURI(uri),
		)

	case strings.ToLower(maxmind.TypeDBIPCountryMMDBIn):
		input = maxmind.NewGeoLite2CountryMMDBIn(
			maxmind.TypeDBIPCountryMMDBIn,
			maxmind.DescDBIPCountryMMDBIn,
			lib.ActionAdd,
			maxmind.WithURI(uri),
		)

	case strings.ToLower(maxmind.TypeIPInfoCountryMMDBIn):
		input = maxmind.NewGeoLite2CountryMMDBIn(
			maxmind.TypeIPInfoCountryMMDBIn,
			maxmind.DescIPInfoCountryMMDBIn,
			lib.ActionAdd,
			maxmind.WithURI(uri),
		)

	case strings.ToLower(mihomo.TypeMRSIn):
		input = mihomo.NewMRSIn(
			lib.ActionAdd,
			mihomo.WithNameAndURI(name, uri),
			mihomo.WithInputDir(dir),
		)

	case strings.ToLower(singbox.TypeSRSIn):
		input = singbox.NewSRSIn(
			lib.ActionAdd,
			singbox.WithNameAndURI(name, uri),
			singbox.WithInputDir(dir),
		)

	case strings.ToLower(v2ray.TypeGeoIPDatIn):
		input = v2ray.NewGeoIPDatIn(
			lib.ActionAdd,
			v2ray.WithURI(uri),
		)

	case strings.ToLower(plaintext.TypeTextIn):
		input = plaintext.NewTextIn(
			plaintext.TypeTextIn,
			plaintext.DescTextIn,
			lib.ActionAdd,
			plaintext.WithNameAndURI(name, uri),
			plaintext.WithInputDir(dir),
		)

	case strings.ToLower(plaintext.TypeClashRuleSetIPCIDRIn):
		input = plaintext.NewTextIn(
			plaintext.TypeClashRuleSetIPCIDRIn,
			plaintext.DescClashRuleSetIPCIDRIn,
			lib.ActionAdd,
			plaintext.WithNameAndURI(name, uri),
			plaintext.WithInputDir(dir),
		)

	case strings.ToLower(plaintext.TypeClashRuleSetClassicalIn):
		input = plaintext.NewTextIn(
			plaintext.TypeClashRuleSetClassicalIn,
			plaintext.DescClashRuleSetClassicalIn,
			lib.ActionAdd,
			plaintext.WithNameAndURI(name, uri),
			plaintext.WithInputDir(dir),
		)

	case strings.ToLower(plaintext.TypeSurgeRuleSetIn):
		input = plaintext.NewTextIn(
			plaintext.TypeSurgeRuleSetIn,
			plaintext.DescSurgeRuleSetIn,
			lib.ActionAdd,
			plaintext.WithNameAndURI(name, uri),
			plaintext.WithInputDir(dir),
		)

	default:
		log.Fatal("unsupported input format")
	}

	return input
}

func getOutputForLookup(search string, searchList ...string) lib.OutputConverter {
	return special.NewLookup(
		lib.ActionOutput,
		special.WithSearch(search),
		special.WithSearchList(searchList),
	)
}
