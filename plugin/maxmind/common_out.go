package maxmind

import (
	"encoding/json"
	"path/filepath"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	defaultOutputName       = "Country.mmdb"
	defaultMaxmindOutputDir = filepath.Join("./", "output", "maxmind")
	defaultDBIPOutputDir    = filepath.Join("./", "output", "db-ip")
)

func newMMDBOut(iType string, iDesc string, action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		OutputName string     `json:"outputName"`
		OutputDir  string     `json:"outputDir"`
		Want       []string   `json:"wantedList"`
		Overwrite  []string   `json:"overwriteList"`
		Exclude    []string   `json:"excludedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.OutputName == "" {
		tmp.OutputName = defaultOutputName
	}

	if tmp.OutputDir == "" {
		switch iType {
		case TypeMaxmindMMDBOut:
			tmp.OutputDir = defaultMaxmindOutputDir

		case TypeDBIPCountryMMDBOut:
			tmp.OutputDir = defaultDBIPOutputDir
		}
	}

	return &MMDBOut{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		OutputName:  tmp.OutputName,
		OutputDir:   tmp.OutputDir,
		Want:        tmp.Want,
		Overwrite:   tmp.Overwrite,
		Exclude:     tmp.Exclude,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}
