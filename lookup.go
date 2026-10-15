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
	strings.ToLower(v2ray.TypeGeoIPDatIn):                  true,
	strings.ToLower(mihomo.TypeMRSIn):                      true,
	strings.ToLower(singbox.TypeSRSIn):                     true,
	strings.ToLower(maxmind.TypeGeoLite2CountryMMDBIn):     true,
	strings.ToLower(maxmind.TypeIPInfoCountryMMDBIn):       true,
	strings.ToLower(maxmind.TypeDBIPCountryMMDBIn):         true,
	strings.ToLower(plaintext.TypeClashRuleSetIPCIDRIn):    true,
	strings.ToLower(plaintext.TypeClashRuleSetClassicalIn): true,
	strings.ToLower(plaintext.TypeSurgeRuleSetIn):          true,
	strings.ToLower(plaintext.TypeTextIn):                  true,
}

var supportedInputFormatsDisallowedDirFlag = map[string]bool{
	strings.ToLower(v2ray.TypeGeoIPDatIn):              true,
	strings.ToLower(maxmind.TypeGeoLite2CountryMMDBIn): true,
	strings.ToLower(maxmind.TypeIPInfoCountryMMDBIn):   true,
	strings.ToLower(maxmind.TypeDBIPCountryMMDBIn):     true,
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

		// Get dir
		// Some formats are single-file formats and do not support directory input
		dir, _ := cmd.Flags().GetString("dir")
		if _, disallowed := supportedInputFormatsDisallowedDirFlag[format]; disallowed && dir != "" {
			log.Fatalf("format %s does not support \"dir\" flag, please use \"uri\" flag instead", format)
		}

		// Set name
		name := "true"

		// Get uri
		uri, _ := cmd.Flags().GetString("uri")

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
		input = &maxmind.GeoLite2CountryMMDBIn{
			Type:        maxmind.TypeGeoLite2CountryMMDBIn,
			Action:      lib.ActionAdd,
			Description: maxmind.DescGeoLite2CountryMMDBIn,
			URI:         uri,
		}

	case strings.ToLower(maxmind.TypeDBIPCountryMMDBIn):
		input = &maxmind.GeoLite2CountryMMDBIn{
			Type:        maxmind.TypeDBIPCountryMMDBIn,
			Action:      lib.ActionAdd,
			Description: maxmind.DescDBIPCountryMMDBIn,
			URI:         uri,
		}

	case strings.ToLower(maxmind.TypeIPInfoCountryMMDBIn):
		input = &maxmind.GeoLite2CountryMMDBIn{
			Type:        maxmind.TypeIPInfoCountryMMDBIn,
			Action:      lib.ActionAdd,
			Description: maxmind.DescIPInfoCountryMMDBIn,
			URI:         uri,
		}

	case strings.ToLower(mihomo.TypeMRSIn):
		input = &mihomo.MRSIn{
			Type:        mihomo.TypeMRSIn,
			Action:      lib.ActionAdd,
			Description: mihomo.DescMRSIn,
			Name:        name,
			URI:         uri,
			InputDir:    dir,
		}

	case strings.ToLower(singbox.TypeSRSIn):
		input = &singbox.SRSIn{
			Type:        singbox.TypeSRSIn,
			Action:      lib.ActionAdd,
			Description: singbox.DescSRSIn,
			Name:        name,
			URI:         uri,
			InputDir:    dir,
		}

	case strings.ToLower(v2ray.TypeGeoIPDatIn):
		input = &v2ray.GeoIPDatIn{
			Type:        v2ray.TypeGeoIPDatIn,
			Action:      lib.ActionAdd,
			Description: v2ray.DescGeoIPDatIn,
			URI:         uri,
		}

	case strings.ToLower(plaintext.TypeTextIn):
		input = &plaintext.TextIn{
			Type:        plaintext.TypeTextIn,
			Action:      lib.ActionAdd,
			Description: plaintext.DescTextIn,
			Name:        name,
			URI:         uri,
			InputDir:    dir,
		}

	case strings.ToLower(plaintext.TypeClashRuleSetIPCIDRIn):
		input = &plaintext.TextIn{
			Type:        plaintext.TypeClashRuleSetIPCIDRIn,
			Action:      lib.ActionAdd,
			Description: plaintext.DescClashRuleSetIPCIDRIn,
			Name:        name,
			URI:         uri,
			InputDir:    dir,
		}

	case strings.ToLower(plaintext.TypeClashRuleSetClassicalIn):
		input = &plaintext.TextIn{
			Type:        plaintext.TypeClashRuleSetClassicalIn,
			Action:      lib.ActionAdd,
			Description: plaintext.DescClashRuleSetClassicalIn,
			Name:        name,
			URI:         uri,
			InputDir:    dir,
		}

	case strings.ToLower(plaintext.TypeSurgeRuleSetIn):
		input = &plaintext.TextIn{
			Type:        plaintext.TypeSurgeRuleSetIn,
			Action:      lib.ActionAdd,
			Description: plaintext.DescSurgeRuleSetIn,
			Name:        name,
			URI:         uri,
			InputDir:    dir,
		}

	default:
		log.Fatal("unsupported input format")
	}

	return input
}

func getOutputForLookup(search string, searchList ...string) lib.OutputConverter {
	return &special.Lookup{
		Type:        special.TypeLookup,
		Action:      lib.ActionOutput,
		Description: special.DescLookup,
		Search:      search,
		SearchList:  searchList,
	}
}
