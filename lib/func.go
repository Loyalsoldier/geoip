package lib

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

var (
	inputConverterMap  = make(map[string]InputConverter)
	outputConverterMap = make(map[string]OutputConverter)
)

func ListInputConverter() {
	fmt.Println("All available input formats:")
	keys := make([]string, 0, len(inputConverterMap))
	for name := range inputConverterMap {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		fmt.Printf("  - %s (%s)\n", name, inputConverterMap[name].GetDescription())
	}
}

func RegisterInputConverter(name string, c InputConverter) error {
	name = strings.TrimSpace(name)
	if _, ok := inputConverterMap[name]; ok {
		return ErrDuplicatedConverter
	}
	inputConverterMap[name] = c
	return nil
}

func ListOutputConverter() {
	fmt.Println("All available output formats:")
	keys := make([]string, 0, len(outputConverterMap))
	for name := range outputConverterMap {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		fmt.Printf("  - %s (%s)\n", name, outputConverterMap[name].GetDescription())
	}
}

func RegisterOutputConverter(name string, c OutputConverter) error {
	name = strings.TrimSpace(name)
	if _, ok := outputConverterMap[name]; ok {
		return ErrDuplicatedConverter
	}
	outputConverterMap[name] = c
	return nil
}

func getRemoteURLContent(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get remote content -> %s: %s", url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}
