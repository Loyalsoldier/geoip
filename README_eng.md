# GeoIP

This project automatically generates GeoIP files every Thursday and provides a command-line interface (CLI) for users to customize their own GeoIP files, including but not limited to V2Ray dat format routing rule file `geoip.dat` and MaxMind mmdb format file `Country.mmdb`.

This project releases GeoIP files automatically every Thursday. It also provides a command line interface(CLI) for users to customize their own GeoIP files, included but not limited to V2Ray dat format file `geoip.dat` and MaxMind mmdb format file `Country.mmdb`.

## Differences from the official version of GeoIP

- The IPv4 address data for Mainland China is a fusion of [IPIP.net](https://github.com/17mon/china_ip_list/blob/master/china_ip_list.txt) and [@gaoyifan/china-operator-ip](https://github.com/gaoyifan/china-operator-ip/blob/ip-lists/china.txt).

- The IPv6 address data for Mainland China is a fusion of MaxMind GeoLite2 and [@gaoyifan/china-operator-ip](https://github.com/gaoyifan/china-operator-ip/blob/ip-lists/china6.txt).

- New categories have been added (to make it easier for users with special requirements to use):
  
  - `geoip:cloudflare`（`GEOIP,CLOUDFLARE`）
  
  - `geoip:cloudfront`（`GEOIP,CLOUDFRONT`）
  
  - `geoip:facebook`（`GEOIP,FACEBOOK`）
  
  - `geoip:fastly`（`GEOIP,FASTLY`）
  
  - `geoip:google`（`GEOIP,GOOGLE`）
  
  - `geoip:netflix`（`GEOIP,NETFLIX`）
  
  - `geoip:telegram`（`GEOIP,TELEGRAM`）
  
  - `geoip:twitter`（`GEOIP,TWITTER`）

## Reference Configuration

The reference configuration for using this project's `.dat` format files in [V2Ray](https://github.com/v2fly/v2ray-core) is：

```json
"routing": {
  "rules": [
    {
      "type": "field",
      "outboundTag": "Direct",
      "ip": [
        "geoip:cn",
        "geoip:private",
        "ext:cn.dat:cn",
        "ext:private.dat:private",
        "ext:geoip-only-cn-private.dat:cn",
        "ext:geoip-only-cn-private.dat:private"
      ]
    },
    {
      "type": "field",
      "outboundTag": "Proxy",
      "ip": [
        "geoip:us",
        "geoip:jp",
        "geoip:facebook",
        "geoip:telegram",
        "ext:geoip-asn.dat:facebook",
        "ext:geoip-asn.dat:telegram"
      ]
    }
  ]
}
```

The reference configuration for using this project's `.mmdb` format files in [Clash](https://github.com/Dreamacro/clash) is:

```yaml
rules:
  - GEOIP,PRIVATE,policy,no-resolve
  - GEOIP,FACEBOOK,policy
  - GEOIP,CN,policy,no-resolve
```

The reference configuration for using this project's `.mmdb` format files in [Leaf](https://github.com/eycorsican/leaf) can be found in the [official README](https://github.com/eycorsican/leaf/blob/master/README.zh.md#geoip).

## Download Link

> If you are unable to access the domain name `raw.githubusercontent.com`, you can use the second address `cdn.jsdelivr.net`.
> *.sha256sum is the verification file.

### V2Ray dat format routing rule file

> Applicable to [V2Ray](https://github.com/v2fly/v2ray-core), [Xray-core](https://github.com/XTLS/Xray-core), and [Trojan-Go](https://github.com/p4gefau1t/trojan-go).

- **geoip.dat**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat)
- **geoip.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat.sha256sum)
- **geoip-only-cn-private.dat**（The simplified version of GeoIP, only includes `geoip:cn` and `geoip:private`）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-only-cn-private.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-only-cn-private.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-only-cn-private.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-only-cn-private.dat)
- **geoip-only-cn-private.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-only-cn-private.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-only-cn-private.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-only-cn-private.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-only-cn-private.dat.sha256sum)
- **geoip-asn.dat**（The simplified version of GeoIP, only includes the above newly added categories）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-asn.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-asn.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-asn.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-asn.dat)
- **geoip-asn.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-asn.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-asn.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-asn.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-asn.dat.sha256sum)
- **cn.dat**（" translates to "The simplified version of GeoIP, only includes `geoip:cn`）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/cn.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/cn.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/cn.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/cn.dat)
- **cn.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/cn.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/cn.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/cn.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/cn.dat.sha256sum)
- **private.dat**（The simplified version of GeoIP, only includes `geoip:private`）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/private.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/private.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/private.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/private.dat)
- **private.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/private.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/private.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/private.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/private.dat.sha256sum)

### MaxMind mmdb format file

> Applicable to [Clash](https://github.com/Dreamacro/clash) and [Leaf](https://github.com/eycorsican/leaf).

- **Country.mmdb**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb)
- **Country.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb.sha256sum)
- **Country-only-cn-private.mmdb**（The streamlined version of GeoIP only contains `GEOIP,CN` and `GEOIP,PRIVATE`）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-only-cn-private.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-only-cn-private.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb)
- **Country-only-cn-private.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-only-cn-private.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-only-cn-private.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb.sha256sum)
- **Country-asn.mmdb**（"GeoIP Lite" that only includes the newly added categories mentioned above）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-asn.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-asn.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-asn.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-asn.mmdb)
- **Country-asn.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-asn.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-asn.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-asn.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-asn.mmdb.sha256sum)

## Customizing GeoIP files

GeoIP files can be customized in the following ways：

- Online Generation: After forking this repository, you can modify the configuration file `config.json` and GitHub Workflow `.github/workflows/build.yml` in your own repository.`
- **local generation**：
  - Install [Golang](https://golang.org/dl/) and [Git](https://git-scm.com/)
  - Clone the project: `git clone https://github.com/Loyalsoldier/geoip.git`
  - Navigate to the root directory of the project: `cd geoip`
  - Modify the configuration file `config.json`
  - Run the code: `go run ./`

**Special note**

- **Online generation**: [Fork](https://github.com/Loyalsoldier/geoip/fork) After this project, if you need to use the MaxMind GeoLite2 Country CSV data file, you need to use **[Settings]* in your warehouse * Add a secret named **MAXMIND_GEOLITE2_LICENSE** to the **[Secrets]** page of the tab, otherwise GitHub Actions will fail to run. The value of this secret is the LICENSE KEY of the MAXMIND account. After [**registering a MAXMIND account**](https://www.maxmind.com/en/geolite2/signup), go to the [**personal account management page**] ](https://www.maxmind.com/en/account) generated in **[My License Key]** under **[Services]** on the left sidebar。
- **Local generation**: If you need to use the MaxMind GeoLite2 Country CSV data file (`GeoLite2-Country-CSV.zip`), you need to download it from MaxMind in advance, or from the project [release branch](https://github.com /Loyalsoldier/geoip/tree/release)[Download](https://github.com/Loyalsoldier/geoip/raw/release/GeoLite2-Country-CSV.zip), and unzip to a directory named `geolite2`.

### Concept Analysis

These two concepts are notable: `input` and `output`. The `input` is the data source and its input format, whereas the `output` is the destination of the converted data and its output format. What the CLI does is to aggregate all input format data, then convert them to output format and write them to GeoIP files by using the options in the config file.

These two concepts are notable: `input` and `output`. The `input` is the data source and its input format, whereas the `output` is the destination of the converted data and its output format. What the CLI does is to aggregate all input format data, then convert them to output format and write them to GeoIP files by using the options in the config file.

### Supported formats include:

About the configuration options supported by each format, check the `config-example.json` file in this project.

Supported `input` formats:

- **text**: Plain text IP and CIDR (e.g., `1.1.1.1` or `1.0.0.0/24`).
- **private**: Local network and private network CIDR (e.g., `192.168.0.0/16` and `127.0.0.0/8`).
- **cutter**: Used to trim data from previous steps.
- **v2rayGeoIPDat**: V2Ray GeoIP dat format (`geoip.dat`).
- **maxmindMMDB**: MaxMind mmdb data format (`GeoLite2-Country.mmdb`).
- **maxmindGeoLite2CountryCSV**: MaxMind GeoLite2 country CSV data (`GeoLite2-Country-CSV.zip`).
- **clashRuleSetClassical**: [Classical type of Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#classical).
- **clashRuleSet**: [ipcidr type of Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#ipcidr).
- **surgeRuleSet**: [Surge RuleSet](https://manual.nssurge.com/rule/ruleset.html).

Supported `output` formats:

- **text**: Plain text CIDR (e.g., `1.0.0.0/24`).
- **v2rayGeoIPDat**: V2Ray GeoIP dat format (`geoip.dat`, suitable for [V2Ray](https://github.com/v2fly/v2ray-core), [Xray-core](https://github.com/XTLS/Xray-core), and [Trojan-Go](https://github.com/p4gefau1t/trojan-go)).
- **maxmindMMDB**: MaxMind mmdb data format (`GeoLite2-Country.mmdb`) used by [Clash](https://github.com/Dreamacro/clash) and [Leaf](https://github.com/eycorsican/leaf).
- **clashRuleSetClassical**: [Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#classical) in classical format.
- **clashRuleSet**: [Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#ipcidr) in `ipcidr` format.
- **surgeRuleSet**: [Surge RuleSet](https://manual.nssurge.com/rule/ruleset.html).

### Considerations

Due to the limitations of the MaxMind mmdb file format, when the IP or CIDR data of different lists have intersections or duplicates, the IP or CIDR data written later will overwrite the data already written in the previous list. For example, the IP `1.1.1.1` belongs to both the `AU` list and the `Cloudflare` list. If `Cloudflare` is written after `AU`, then the IP `1.1.1.1` belongs to the `Cloudflare` list.

To ensure that a specified list or a modified list always includes all of its own IP or CIDR data, you can add the option `overwriteList` in the configuration with `output` format set to `maxmindMMDB`. The lists specified in the `overwriteList` option will be written one by one at the end, with the last item in the list having the highest priority. If the `wantedList` option has been set, there is no need to set the `overwriteList` option. The lists specified in `wantedList` will be written one by one at the end, with the last item in the list having the highest priority.

## CLI Functionality Showcase

You can directly install the CLI by running `go install -v github.com/Loyalsoldier/geoip@latest`.

```bash
$ ./geoip -h
Usage of ./geoip:
  -c string
        URI of the JSON format config file, support both local file path and remote HTTP(S) URL (default "config.json")
  -l    List all available input and output formats

$ ./geoip -c config.json
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] geoip.dat --> output/dat
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] geoip-only-cn-private.dat --> output/dat
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] geoip-asn.dat --> output/dat
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] cn.dat --> output/dat
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] private.dat --> output/dat
2021/08/29 12:11:39 ✅ [maxmindMMDB] Country.mmdb --> output/maxmind
2021/08/29 12:11:39 ✅ [maxmindMMDB] Country-only-cn-private.mmdb --> output/maxmind
2021/08/29 12:11:39 ✅ [text] netflix.txt --> output/text
2021/08/29 12:11:39 ✅ [text] telegram.txt --> output/text
2021/08/29 12:11:39 ✅ [text] cn.txt --> output/text
2021/08/29 12:11:39 ✅ [text] cloudflare.txt --> output/text
2021/08/29 12:11:39 ✅ [text] cloudfront.txt --> output/text
2021/08/29 12:11:39 ✅ [text] facebook.txt --> output/text
2021/08/29 12:11:39 ✅ [text] fastly.txt --> output/text

$ ./geoip -l
All available input formats:
  - v2rayGeoIPDat (Convert V2Ray GeoIP dat to other formats)
  - maxmindMMDB (Convert MaxMind mmdb database to other formats)
  - maxmindGeoLite2CountryCSV (Convert MaxMind GeoLite2 country CSV data to other formats)
  - private (Convert LAN and private network CIDR to other formats)
  - text (Convert plaintext IP & CIDR to other formats)
  - clashRuleSetClassical (Convert classical type of Clash RuleSet to other formats (just processing IP & CIDR lines))
  - clashRuleSet (Convert ipcidr type of Clash RuleSet to other formats)
  - surgeRuleSet (Convert Surge RuleSet to other formats (just processing IP & CIDR lines))
  - cutter (Remove data from previous steps)
  - test (Convert specific CIDR to other formats (for test only))
All available output formats:
  - v2rayGeoIPDat (Convert data to V2Ray GeoIP dat format)
  - maxmindMMDB (Convert data to MaxMind mmdb database format)
  - clashRuleSetClassical (Convert data to classical type of Clash RuleSet)
  - clashRuleSet (Convert data to ipcidr type of Clash RuleSet)
  - surgeRuleSet (Convert data to Surge RuleSet)
  - text (Convert data to plaintext CIDR format)
```

## License

[CC-BY-SA-4.0](https://creativecommons.org/licenses/by-sa/4.0/)

This product includes GeoLite2 data created by MaxMind, available from [MaxMind](http://www.maxmind.com).
