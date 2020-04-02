// GeoIP generator
//
// Before running this file, the GeoIP database must be downloaded and present.
// To download GeoIP database: https://dev.maxmind.com/geoip/geoip2/geolite2/
// Inside you will find block files for IPv4 and IPv6 and country code mapping.
package main

import (
	"compress/gzip"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"v2ray.com/core/app/router"
	"v2ray.com/core/common"
	"v2ray.com/core/infra/conf"
)

var (
	countryCodeFile = flag.String("country", "", "Path to the country code file")
	ipv4File        = flag.String("ipv4", "", "Path to the IPv4 block file")
	ipv6File        = flag.String("ipv6", "", "Path to the IPv6 block file")
	cnIPv4CIDRURL   = flag.String("cnipv4url", "", "URL of CN IPv4 CIDR file")
)

func getCountryCodeMap() (map[string]string, error) {
	countryCodeReader, err := os.Open(*countryCodeFile)
	if err != nil {
		return nil, err
	}
	defer countryCodeReader.Close()

	m := make(map[string]string)
	reader := csv.NewReader(countryCodeReader)
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	for _, line := range lines[1:] {
		id := line[0]
		countryCode := line[4]
		if len(countryCode) == 0 {
			continue
		}
		m[id] = strings.ToUpper(countryCode)
	}
	return m, nil
}

func getCidrPerCountry(file string, m map[string]string, list map[string][]*router.CIDR) error {
	fileReader, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	reader := csv.NewReader(fileReader)
	lines, err := reader.ReadAll()
	if err != nil {
		return err
	}
	for _, line := range lines[1:] {
		cidrStr := line[0]
		countryID := line[1]
		if countryCode, found := m[countryID]; found {
			cidr, err := conf.ParseIP(cidrStr)
			if err != nil {
				return err
			}
			cidrs := append(list[countryCode], cidr)
			list[countryCode] = cidrs
		}
	}
	return nil
}

func getCNIPv4Cidr(url string) (cnIPv4CidrList []string, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer data.Close()

	body, err := ioutil.ReadAll(data)
	if err != nil {
		return nil, err
	}

	cnIPv4CidrList = strings.Split(string(body), "\n")
	fmt.Println("The length of cnIPv4CIDRList is", len(cnIPv4CidrList))

	return cnIPv4CidrList, nil
}

func changeCNIPv4Cidr(url string, m map[string]string, list map[string][]*router.CIDR) error {
	// delete "CN" Key in list
	delete(list, "CN")
	fmt.Println("Successfully deleted CN IPv4 CIDR")
	fmt.Println(list["CN"])

	cnIPv4CidrList, err := getCNIPv4Cidr(url)
	if err != nil {
		return err
	}

	for _, cnIPv4Cidr := range cnIPv4CidrList {
		cnIPv4Cidr, err := conf.ParseIP(strings.TrimSpace(cnIPv4Cidr))
		if err != nil {
			return err
		}
		cidrs := append(list["CN"], cnIPv4Cidr)
		list["CN"] = cidrs
	}

	return nil
}

func main() {
	flag.Parse()

	ccMap, err := getCountryCodeMap()
	if err != nil {
		fmt.Println("Error reading country code map:", err)
		return
	}

	cidrList := make(map[string][]*router.CIDR)
	if err := getCidrPerCountry(*ipv4File, ccMap, cidrList); err != nil {
		fmt.Println("Error loading IPv4 file:", err)
		return
	}
	if err := changeCNIPv4Cidr(*cnIPv4CIDRURL, ccMap, cidrList); err != nil {
		fmt.Println("Error loading cnIPv4CIDR data:", err)
		return
	}
	if err := getCidrPerCountry(*ipv6File, ccMap, cidrList); err != nil {
		fmt.Println("Error loading IPv6 file:", err)
		return
	}

	geoIPList := new(router.GeoIPList)
	for cc, cidr := range cidrList {
		geoIPList.Entry = append(geoIPList.Entry, &router.GeoIP{
			CountryCode: cc,
			Cidr:        cidr,
		})
	}
	geoIPList.Entry = append(geoIPList.Entry, getLocalIPs())

	geoIPBytes, err := proto.Marshal(geoIPList)
	if err != nil {
		fmt.Println("Error marshalling geoip list:", err)
	}

	if err := ioutil.WriteFile("geoip.dat", geoIPBytes, 0777); err != nil {
		fmt.Println("Error writing geoip to file:", err)
	}
}

var (
	localIPs = []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.88.99.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
)

func getLocalIPs() *router.GeoIP {
	cidr := make([]*router.CIDR, 0, 16)
	for _, ip := range localIPs {
		c, err := conf.ParseIP(ip)
		common.Must(err)
		cidr = append(cidr, c)
	}
	return &router.GeoIP{
		CountryCode: "PRIVATE",
		Cidr:        cidr,
	}
}
