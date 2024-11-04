# 简介

本项目每周四自动生成多种格式 GeoIP 文件，同时提供命令行界面（CLI）工具供用户自行定制 GeoIP 文件，包括但不限于 V2Ray `dat` 格式文件 `geoip.dat`、MaxMind `mmdb` 格式文件 `Country.mmdb`、sing-box `SRS` 格式文件、mihomo `MRS` 格式文件、Clash ruleset 和 Surge ruleset。

This project releases various formats of GeoIP files automatically every Thursday, and provides a command line interface(CLI) tool for users to customize their own GeoIP files, including but not limited to V2Ray `dat` format file `geoip.dat`, MaxMind `mmdb` format file `Country.mmdb`, sing-box `SRS` format files, mihomo `MRS` format files, Clash ruleset files and Surge ruleset files.

## 与 MaxMind 官方 GeoIP 数据的区别

本项目默认使用 [MaxMind GeoLite2 Country CSV 数据](https://github.com/Loyalsoldier/geoip/blob/release/GeoLite2-Country-CSV.zip)生成各个国家和地区的 GeoIP 文件。所有可供使用的国家和地区 geoip 类别（如 `geoip:cn`，两位英文字母表示国家和地区），请查看：[https://www.iban.com/country-codes](https://www.iban.com/country-codes)。

另外，本项目对 MaxMind 官方 GeoIP 数据做了修改和新增：

- 中国大陆 IPv4 地址数据融合了 [IPIP.net](https://github.com/17mon/china_ip_list/blob/master/china_ip_list.txt) 和 [@gaoyifan/china-operator-ip](https://github.com/gaoyifan/china-operator-ip/blob/ip-lists/china.txt)
- 中国大陆 IPv6 地址数据融合了 MaxMind GeoLite2 和 [@gaoyifan/china-operator-ip](https://github.com/gaoyifan/china-operator-ip/blob/ip-lists/china6.txt)
- 新增类别（方便有特殊需求的用户使用）：
  - `geoip:cloudflare`（`GEOIP,CLOUDFLARE`）
  - `geoip:cloudfront`（`GEOIP,CLOUDFRONT`）
  - `geoip:facebook`（`GEOIP,FACEBOOK`）
  - `geoip:fastly`（`GEOIP,FASTLY`）
  - `geoip:google`（`GEOIP,GOOGLE`）
  - `geoip:netflix`（`GEOIP,NETFLIX`）
  - `geoip:telegram`（`GEOIP,TELEGRAM`）
  - `geoip:twitter`（`GEOIP,TWITTER`）
  - `geoip:tor`（`GEOIP,TOR`）

## 下载地址与使用方法

本项目发布的所有 GeoIP 文件，请查看 [release 分支](https://github.com/Loyalsoldier/geoip/tree/release)。以下是部分格式 GeoIP 文件的下载地址：

> 如果无法访问域名 `raw.githubusercontent.com`，可以使用第二个地址 `cdn.jsdelivr.net`。
> 如果无法访问域名 `cdn.jsdelivr.net`，可以将其替换为 `fastly.jsdelivr.net`。
>
> *.sha256sum 为校验文件。

### V2Ray dat 格式文件

> 适用于 [V2Ray](https://github.com/v2fly/v2ray-core)、[Xray-core](https://github.com/XTLS/Xray-core)、[mihomo](https://github.com/MetaCubeX/mihomo/tree/Meta)、[hysteria](https://github.com/apernet/hysteria)、[Trojan-Go](https://github.com/p4gefau1t/trojan-go)。

> 此 dat 格式文件不能用于 Nginx。

- **geoip.dat**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat)
- **geoip.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat.sha256sum)
- **geoip-only-cn-private.dat**（精简版 GeoIP，只包含 `geoip:cn` 和 `geoip:private`）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-only-cn-private.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-only-cn-private.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-only-cn-private.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-only-cn-private.dat)
- **geoip-only-cn-private.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-only-cn-private.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-only-cn-private.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-only-cn-private.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-only-cn-private.dat.sha256sum)
- **geoip-asn.dat**（精简版 GeoIP，只包含上述新增类别）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-asn.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-asn.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-asn.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-asn.dat)
- **geoip-asn.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-asn.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip-asn.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-asn.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip-asn.dat.sha256sum)
- **cn.dat**（精简版 GeoIP，只包含 `geoip:cn`）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/cn.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/cn.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/cn.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/cn.dat)
- **cn.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/cn.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/cn.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/cn.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/cn.dat.sha256sum)
- **private.dat**（精简版 GeoIP，只包含 `geoip:private`）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/private.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/private.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/private.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/private.dat)
- **private.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/private.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/private.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/private.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/private.dat.sha256sum)
- **所有国家 / 地区 / 新增类别**的 dat 格式文件，请查看本项目 `release` 分支下的 [dat 目录](https://github.com/Loyalsoldier/geoip/tree/release/dat)。

#### dat 格式文件使用方法

<details>
  <summary>点击查看在 <b>V2Ray</b> 和 <b>Xray-core</b> 中的使用方法</summary>
  <br/>
  <p>需要先下载 <code>.dat</code> 格式文件，并放置在程序目录内。</p>

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
</details>

<details>
  <summary>点击查看在 <b>mihomo</b> 中的使用方法</summary>

```yaml
geodata-mode: true
geox-url:
  geoip: "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat"
```
</details>

<details>
  <summary>点击查看在 <b>hysteria</b> 中的使用方法</summary>
  <br/>
  <p>需要先下载 <code>.dat</code> 格式文件，并放置在 hysteria 程序目录内。</p>

```
direct(geoip:cn)
proxy(geoip:telegram)
proxy(geoip:us)
```
</details>

<details>
  <summary>点击查看在 <b>Trojan-Go</b> 中的使用方法</summary>
  <br/>
  <p>需要先下载 <code>.dat</code> 格式文件，并放置在 Trojan-Go 程序目录内。</p>

```json
"router": {
  "enabled": true,
  "bypass": ["geoip:cn"],
  "proxy": ["geoip:telegram", "geoip:us"],
  "block": ["geoip:jp"],
  "default_policy": "proxy",
  "geoip": "./geoip.dat"
}
```
</details>

---

### MaxMind mmdb 格式文件

MaxMind 官方版**国家/地区**类型 mmdb 文件：

> 适用于 [Clash](https://github.com/Dreamacro/clash)、[mihomo](https://github.com/MetaCubeX/mihomo/tree/Meta)、[Shadowrocket](https://apps.apple.com/us/app/id932747118)、[Quantumult X](https://apps.apple.com/us/app/id1443988620)、[Surge](https://nssurge.com)、[Leaf](https://github.com/eycorsican/leaf)。

> 适用于 [Nginx](https://nginx.org)，需要配合 [ngx_http_geoip2_module](https://github.com/leev/ngx_http_geoip2_module) 模块使用。

- **GeoLite2-Country.mmdb**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/GeoLite2-Country.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/GeoLite2-Country.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-Country.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-Country.mmdb)
- **GeoLite2-Country.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/GeoLite2-Country.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/GeoLite2-Country.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-Country.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-Country.mmdb.sha256sum)

MaxMind 官方版 **ASN** 类型 mmdb 文件：

> 适用于 [mihomo](https://github.com/MetaCubeX/mihomo/tree/Meta)、[Shadowrocket](https://apps.apple.com/us/app/id932747118)、[Surge](https://nssurge.com)。

- **GeoLite2-ASN.mmdb**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/GeoLite2-ASN.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/GeoLite2-ASN.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-ASN.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-ASN.mmdb)
- **GeoLite2-ASN.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/GeoLite2-ASN.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/GeoLite2-ASN.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-ASN.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-ASN.mmdb.sha256sum)

本项目生成的**国家/地区**类型 mmdb 文件：

> 适用于 [Clash](https://github.com/Dreamacro/clash)、[mihomo](https://github.com/MetaCubeX/mihomo/tree/Meta)、[Shadowrocket](https://apps.apple.com/us/app/id932747118)、[Quantumult X](https://apps.apple.com/us/app/id1443988620)、[Surge](https://nssurge.com)、[Leaf](https://github.com/eycorsican/leaf)。

> 适用于 [Nginx](https://nginx.org)，需要配合 [ngx_http_geoip2_module](https://github.com/leev/ngx_http_geoip2_module) 模块使用。

> **国家/地区**类别保留了 `Continent` 和 `Country` 里的所有字段。**新增类别**和 **geoip:private** 类别只保留了 `Country` 里的 `iso_code`（两位英文字母表示的国家/地区代号）字段。关于 Maxmind 官方 country MMDB 格式文件完整字段，请[查看代码](https://github.com/oschwald/geoip2-golang/blob/576a46d19bb59f32d0215cb43285b8928891b6bc/reader.go#L139-L171)。

- **Country-without-asn.mmdb**（传统版 GeoIP，只包含国家/地区类别，不包含上述新增类别。建议优先使用）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-without-asn.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-without-asn.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-without-asn.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-without-asn.mmdb)
- **Country-without-asn.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-without-asn.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-without-asn.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-without-asn.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-without-asn.mmdb.sha256sum)
- **Country.mmdb**（增强版 GeoIP，包含国家/地区类别，以及上述新增类别。但由于 MaxMind mmdb 格式限制，部分国家/地区类别的 IP 地址数据不如上述 **Country-without-asn.mmdb** 准确）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb)
- **Country.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb.sha256sum)
- **Country-only-cn-private.mmdb**（精简版 GeoIP，只包含 `GEOIP,CN` 和 `GEOIP,PRIVATE`）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-only-cn-private.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-only-cn-private.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb)
- **Country-only-cn-private.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-only-cn-private.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-only-cn-private.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb.sha256sum)
- **Country-asn.mmdb**（精简版 GeoIP，只包含上述新增类别）：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-asn.mmdb](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-asn.mmdb)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-asn.mmdb](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-asn.mmdb)
- **Country-asn.mmdb.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-asn.mmdb.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country-asn.mmdb.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-asn.mmdb.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-asn.mmdb.sha256sum)

#### mmdb 格式文件使用方法

<details>
  <summary>点击查看在 <b>Clash</b> 中的使用方法</summary>
  <br/>
  <p>需要先下载 <code>.mmdb</code> 格式文件，命名为 <code>Country.mmdb</code>，并放置在 Clash 程序目录内。</p>

```yaml
rules:
  - GEOIP,PRIVATE,policy,no-resolve
  - GEOIP,FACEBOOK,policy
  - GEOIP,CN,policy,no-resolve
```
</details>

<details>
  <summary>点击查看在 <b>mihomo</b> 中的使用方法</summary>

```yaml
geodata-mode: true
geox-url:
  mmdb: "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb"
  asn: "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/GeoLite2-ASN.mmdb"
```
</details>

<details>
  <summary>点击查看在 <b>Shadowrocket</b> 中的使用方法</summary>
  <br/>
  <p>需要将下载地址填入到 Shadowrocket 的设置中。</p>

```conf
[Rule]
GEOIP,PRIVATE,DIRECT
GEOIP,FACEBOOK,PROXY
GEOIP,CN,DIRECT
```
</details>

<details>
  <summary>点击查看在 <b>Quantumult X</b> 中的使用方法</summary>
  <br/>
  <p>需要将下载地址填入到 Quantumult X 的设置中。</p>

```conf
[filter_local]
GEOIP,PRIVATE,DIRECT
GEOIP,FACEBOOK,PROXY
GEOIP,CN,DIRECT
```
</details>

<details>
  <summary>点击查看在 <b>Surge</b> 中的使用方法</summary>
  <br/>
  <p>需要将下载地址填入到 Surge 的设置中。</p>

```conf
[Rule]
GEOIP,PRIVATE,policy,no-resolve
GEOIP,FACEBOOK,policy
GEOIP,CN,policy,no-resolve
```
</details>

---

### sing-box SRS 格式文件

> 适用于 [sing-box](https://github.com/SagerNet/sing-box)。

请查看本项目 `release` 分支下的 [srs 目录](https://github.com/Loyalsoldier/geoip/tree/release/srs)。

#### SRS 格式文件使用方法

<details>
  <summary>点击查看在 <b>sing-box</b> 中的使用方法</summary>

```json
"route": {
  "rules": [
    {
      "rule_set": "geoip-cn",
      "outbound": "direct"
    },
    {
      "rule_set": "geoip-us",
      "outbound": "block"
    }
  ],
  "rule_set": [
    {
      "tag": "geoip-cn",
      "type": "remote",
      "format": "binary",
      "url": "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/srs/cn.srs"
    },
    {
      "tag": "geoip-us",
      "type": "remote",
      "format": "binary",
      "url": "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/srs/us.srs"
    }
  ]
}
```
</details>

---

### mihomo MRS 格式文件

> 适用于 [mihomo](https://github.com/MetaCubeX/mihomo/tree/Meta)。

请查看本项目 `release` 分支下的 [mrs 目录](https://github.com/Loyalsoldier/geoip/tree/release/mrs)。

#### MRS 格式文件使用方法

<details>
  <summary>点击查看在 <b>mihomo</b> 中的使用方法</summary>

```yaml
rule-providers:
  cn-cidr:
    type: http
    behavior: ipcidr
    format: mrs
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/mrs/cn.mrs"
    path: ./mrs/geoip/cn.mrs
    interval: 86400

  google-cidr:
    type: http
    behavior: ipcidr
    format: mrs
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/mrs/google.mrs"
    path: ./mrs/geoip/google.mrs
    interval: 86400

rules:
  - RULE-SET,cn-cidr,DIRECT
  - RULE-SET,google-cidr,PROXY,no-resolve
```
</details>

---

### Clash ruleset 文件

> 适用于 [Clash Premium](https://github.com/Dreamacro/clash)、[mihomo](https://github.com/MetaCubeX/mihomo/tree/Meta)。

请查看本项目 `release` 分支下的 [clash 目录](https://github.com/Loyalsoldier/geoip/tree/release/clash)。

#### Clash ruleset 使用方法

<details>
  <summary>点击查看在 <b>Clash Premium</b> 和 <b>mihomo</b> 中的使用方法</summary>

```yaml
rule-providers:
  cn-cidr:
    type: http
    behavior: ipcidr
    format: yaml
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/clash/ipcidr/cn.txt"
    path: ./ruleset/ipcidr/cn.yaml
    interval: 86400

  telegram-cidr:
    type: http
    behavior: ipcidr
    format: yaml
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/clash/ipcidr/telegram.txt"
    path: ./ruleset/ipcidr/telegram.yaml
    interval: 86400

rules:
  - RULE-SET,cn-cidr,DIRECT
  - RULE-SET,telegram-cidr,PROXY,no-resolve
```
</details>

---

### Surge ruleset 文件

> 适用于 [Surge](https://nssurge.com)。

请查看本项目 `release` 分支下的 [surge 目录](https://github.com/Loyalsoldier/geoip/tree/release/surge)。

#### Surge ruleset 使用方法

<details>
  <summary>点击查看在 <b>Surge</b> 中的使用方法</summary>

```conf
[Rule]
RULE-SET,https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/surge/us.txt,REJECT
RULE-SET,https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/surge/cn.txt,DIRECT
RULE-SET,https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/surge/telegram.txt,PROXY,no-resolve
```
</details>

---

### 纯文本 txt 格式文件

请查看本项目 `release` 分支下的 [text 目录](https://github.com/Loyalsoldier/geoip/tree/release/text)。

---

### Nginx `allow` 和 `deny` 文件

请查看本项目 `release` 分支下的 [nginx 目录](https://github.com/Loyalsoldier/geoip/tree/release/nginx)。

---

## 自行定制 GeoIP 文件

可通过以下几种方式自行定制 GeoIP 文件：

- **在线生成**：[Fork](https://github.com/Loyalsoldier/geoip/fork) 本仓库后，修改自己仓库内的配置文件 `config.json` 和 GitHub Workflow `.github/workflows/build.yml`
- **本地生成**：
  - 安装 [Golang](https://go.dev/dl/) 和 [Git](https://git-scm.com)
  - 拉取项目代码: `git clone https://github.com/Loyalsoldier/geoip.git`
  - 进入项目根目录：`cd geoip`
  - 修改配置文件 `config.json`
  - 运行代码：`go run ./ convert -c ./config.json`

**特别说明：**

- **在线生成**：[Fork](https://github.com/Loyalsoldier/geoip/fork) 本项目后，如果需要使用 MaxMind GeoLite2 官方数据文件，需要在自己仓库的 **[Settings]** 页面的左侧边栏 **[Secrets and variables]** 下的 **[Actions]** 选项卡页面中添加一个名为 **MAXMIND_GEOLITE2_LICENSE** 的 secret，否则 GitHub Actions 会运行失败。这个 secret 的值为 MaxMind 账号的 LICENSE KEY，需要[**注册 MaxMind 账号**](https://www.maxmind.com/en/geolite2/signup)后，在[**个人账号管理页面**](https://www.maxmind.com/en/account)左侧边栏的 [**Manage License Keys**] 里生成。
- **本地生成**：如果需要使用 MaxMind 官方 GeoLite2 数据文件，需要提前从 MaxMind 下载，或者从本项目 [release 分支](https://github.com/Loyalsoldier/geoip/tree/release)下载（文件名以 `GeoLite2` 为前缀的文件），并解压缩到名为 `geolite2` 的目录。

### 概念解析

本项目有两个概念：`input` 和 `output`。`input` 指数据源（data source）及其输入格式，`output` 指数据的去向（data destination）及其输出格式。CLI 的作用就是通过读取配置文件中的选项，聚合用户提供的所有数据源，去重，将其转换为目标格式，并输出到文件。

These two concepts are notable: `input` and `output`. The `input` is the data source and its input format, whereas the `output` is the destination of the converted data and its output format. What the CLI does is to aggregate all input format data, then convert them to output format and write them to GeoIP files by using the options in the config file.

### 支持的格式

关于每种格式所支持的配置选项，查看本项目 [`configuration.md`](https://github.com/Loyalsoldier/geoip/blob/HEAD/configuration.md) 文件。

支持的 `input` 输入格式：

- **text**：纯文本 IP 和 CIDR（例如：`1.1.1.1` 或 `1.0.0.0/24`）
- **stdin**：从 standard input 获取纯文本 IP 和 CIDR（例如：`1.1.1.1` 或 `1.0.0.0/24`）
- **private**：局域网和私有网络 CIDR（例如：`192.168.0.0/16` 和 `127.0.0.0/8`）
- **cutter**：用于裁剪前置步骤中的数据
- **json**：JSON 数据格式
- **v2rayGeoIPDat**：V2Ray GeoIP dat 数据格式（`geoip.dat`）
- **maxmindMMDB**：MaxMind GeoLite2 country mmdb 数据格式（`GeoLite2-Country.mmdb`）
- **maxmindGeoLite2ASNCSV**：MaxMind GeoLite2 ASN CSV 数据格式（`GeoLite2-ASN-CSV.zip`）
- **maxmindGeoLite2CountryCSV**：MaxMind GeoLite2 country CSV 数据格式（`GeoLite2-Country-CSV.zip`）
- **dbipCountryMMDB**：DB-IP country mmdb 数据格式（`dbip-country-lite.mmdb`）
- **ipinfoCountryMMDB**：IPInfo country mmdb 数据格式（`country.mmdb`）
- **mihomoMRS**：mihomo MRS 数据格式（`geoip-cn.mrs`）
- **singboxSRS**：sing-box SRS 数据格式（`geoip-cn.srs`）
- **clashRuleSetClassical**：[classical 类型的 Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#classical)
- **clashRuleSet**：[ipcidr 类型的 Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#ipcidr)
- **surgeRuleSet**：[Surge RuleSet](https://manual.nssurge.com/rule/ruleset.html)

支持的 `output` 输出格式：

- **text**：纯文本 CIDR（例如：`1.0.0.0/24`）
- **stdout**：将纯文本 CIDR 输出到 standard output（例如：`1.0.0.0/24`）
- **lookup**：从指定的列表中查找指定的 IP 或 CIDR
- **v2rayGeoIPDat**：V2Ray GeoIP dat 数据格式（`geoip.dat`）
- **maxmindMMDB**：MaxMind GeoLite2 country mmdb 数据格式（`GeoLite2-Country.mmdb`）
- **dbipCountryMMDB**：DB-IP country mmdb 数据格式（`dbip-country-lite.mmdb`）
- **ipinfoCountryMMDB**：IPInfo country mmdb 数据格式（`country.mmdb`）
- **mihomoMRS**：mihomo MRS 数据格式（`geoip-cn.mrs`）
- **singboxSRS**：sing-box SRS 数据格式（`geoip-cn.srs`）
- **clashRuleSetClassical**：[classical 类型的 Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#classical)
- **clashRuleSet**：[ipcidr 类型的 Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#ipcidr)
- **surgeRuleSet**：[Surge RuleSet](https://manual.nssurge.com/rule/ruleset.html)

### 注意事项

由于 MaxMind、DB-IP、IPInfo 的 mmdb 文件格式的限制，当不同列表的 IP 或 CIDR 数据有交集或重复项时，后写入的列表的 IP 或 CIDR 数据会覆盖（overwrite）之前已写入的列表的数据。譬如，IP `1.1.1.1` 同属于列表 `AU` 和列表 `Cloudflare`。如果 `Cloudflare` 在 `AU` 之后写入，则 IP `1.1.1.1` 归属于列表 `Cloudflare`。

为了确保某些指定的列表、被修改的列表一定囊括属于它的所有 IP 或 CIDR 数据，可在 `output` 相应输出格式的配置中增加选项 `overwriteList`，该选项中指定的列表会在最后逐一写入，列表中最后一项优先级最高。若已设置选项 `wantedList`，则无需设置 `overwriteList`。`wantedList` 中指定的列表会在最后逐一写入，列表中最后一项优先级最高。

## CLI 功能展示

可通过 `go install -v github.com/Loyalsoldier/geoip@latest` 直接安装 CLI 工具。

CLI 提供的功能如下：

- 列出支持的 `input` 和 `output` 格式（`list`）
- GeoIP 数据格式转换（`convert`）
- 查找 IP 或 CIDR 所在类别（`lookup`）
- 去重和合并 IP 与 CIDR（`merge`）

### 总览

```bash
$ ./geoip
geoip is a convenient tool to merge, convert and lookup IP & CIDR from various formats of geoip data.

Usage:
  geoip [command]

Available Commands:
  convert     Convert geoip data from one format to another by using config file
  help        Help about any command
  list        List all available input and output formats
  lookup      Lookup specified IP or CIDR in specified lists
  merge       Merge plaintext IP & CIDR from standard input, then print to standard output

Flags:
  -h, --help   help for geoip

Use "geoip [command] --help" for more information about a command.
```

### 列出支持的 `input` 和 `output` 格式（`list`）

```bash
$ ./geoip list
All available input formats:
  - clashRuleSet (Convert ipcidr type of Clash RuleSet to other formats)
  - clashRuleSetClassical (Convert classical type of Clash RuleSet to other formats (just processing IP & CIDR lines))
  - cutter (Remove data from previous steps)
  - dbipCountryMMDB (Convert DB-IP country mmdb database to other formats)
  - ipinfoCountryMMDB (Convert IPInfo country mmdb database to other formats)
  - json (Convert JSON data to other formats)
  - maxmindGeoLite2ASNCSV (Convert MaxMind GeoLite2 ASN CSV data to other formats)
  - maxmindGeoLite2CountryCSV (Convert MaxMind GeoLite2 country CSV data to other formats)
  - maxmindMMDB (Convert MaxMind mmdb database to other formats)
  - mihomoMRS (Convert mihomo MRS data to other formats)
  - private (Convert LAN and private network CIDR to other formats)
  - singboxSRS (Convert sing-box SRS data to other formats)
  - stdin (Accept plaintext IP & CIDR from standard input, separated by newline)
  - surgeRuleSet (Convert Surge RuleSet to other formats (just processing IP & CIDR lines))
  - test (Convert specific CIDR to other formats (for test only))
  - text (Convert plaintext IP & CIDR to other formats)
  - v2rayGeoIPDat (Convert V2Ray GeoIP dat to other formats)

All available output formats:
  - clashRuleSet (Convert data to ipcidr type of Clash RuleSet)
  - clashRuleSetClassical (Convert data to classical type of Clash RuleSet)
  - dbipCountryMMDB (Convert data to DB-IP country mmdb database format)
  - ipinfoCountryMMDB (Convert data to IPInfo country mmdb database format)
  - lookup (Lookup specified IP or CIDR from various formats of data)
  - maxmindMMDB (Convert data to MaxMind mmdb database format)
  - mihomoMRS (Convert data to mihomo MRS format)
  - singboxSRS (Convert data to sing-box SRS format)
  - stdout (Convert data to plaintext CIDR format and output to standard output)
  - surgeRuleSet (Convert data to Surge RuleSet)
  - text (Convert data to plaintext CIDR format)
  - v2rayGeoIPDat (Convert data to V2Ray GeoIP dat format)
```

### 去重和合并 IP 与 CIDR（`merge`）

```bash
$ curl -s https://core.telegram.org/resources/cidr.txt | ./geoip merge -t ipv4
91.105.192.0/23
91.108.4.0/22
91.108.8.0/21
91.108.16.0/21
91.108.56.0/22
149.154.160.0/20
185.76.151.0/24
```

### GeoIP 数据格式转换（`convert`）

```bash
$ ./geoip convert -c config.json
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
2021/08/29 12:11:45 ✅ [singboxSRS] netflix.txt --> output/srs
2021/08/29 12:11:45 ✅ [singboxSRS] telegram.txt --> output/srs
2021/08/29 12:11:45 ✅ [singboxSRS] cn.txt --> output/srs
2021/08/29 12:11:45 ✅ [singboxSRS] cloudflare.txt --> output/srs
2021/08/29 12:11:45 ✅ [singboxSRS] cloudfront.txt --> output/srs
2021/08/29 12:11:45 ✅ [singboxSRS] facebook.txt --> output/srs
2021/08/29 12:11:45 ✅ [singboxSRS] fastly.txt --> output/srs
2021/08/29 12:11:50 ✅ [mihomoMRS] netflix.txt --> output/mrs
2021/08/29 12:11:50 ✅ [mihomoMRS] telegram.txt --> output/mrs
2021/08/29 12:11:50 ✅ [mihomoMRS] cn.txt --> output/mrs
2021/08/29 12:11:50 ✅ [mihomoMRS] cloudflare.txt --> output/mrs
2021/08/29 12:11:50 ✅ [mihomoMRS] cloudfront.txt --> output/mrs
2021/08/29 12:11:50 ✅ [mihomoMRS] facebook.txt --> output/mrs
2021/08/29 12:11:50 ✅ [mihomoMRS] fastly.txt --> output/mrs
```

### 查找 IP 或 CIDR 所在类别（`lookup`）

可能的返回结果：

- 查询的字符串不是有效的 IP 或 CIDR，返回 `false`
- 查询的 IP 或 CIDR 不存在于任何一个类别中，返回 `false`
- 查询的 IP 或 CIDR 存在于某种格式文件的单个类别中：
  - 若该格式文件只包含一个类别，返回 `true`
  - 若该格式文件包含多个类别，返回匹配的类别名称
- 查询的 IP 或 CIDR 存在于多个类别中，返回以英文逗号分隔的类别名称，如 `au,cloudflare`

```bash
# ================= One-time Mode ================= #

# 从 text 格式的本地文件（只包含一个类别）中查找某个 IP 地址
# lookup IP from local file (with only one list) in text format
$ ./geoip lookup -f text -u ./cn.txt 1.0.1.1
true


# 从 text 格式的本地文件（只包含一个类别）中查找某个 IP 地址
# lookup IP from local file (with only one list) in text format
$ ./geoip lookup -f text -u ./cn.txt 2.2.2.2
false


# 从 text 格式的本地文件（只包含一个类别）中查找某个 CIDR
# lookup CIDR from local file (with only one list) in text format
$ ./geoip lookup -f text -u ./cn.txt 1.0.1.1/24
true


# 从 text 格式的本地文件（只包含一个类别）中查找某个 CIDR
# lookup CIDR from local file (with only one list) in text format
$ ./geoip lookup -f text -u ./cn.txt 1.0.1.1/23
false


# 从 text 格式的远程 URL（只包含一个类别）中查找某个 IP 地址
# lookup IP from remote URL (with only one list) in text format
$ ./geoip lookup -f text -u https://example.com/cn.txt 1.0.1.1
true


# 从 v2rayGeoIPDat 格式的本地文件（只包含一个类别）中查找某个 IP 地址
# lookup IP from local file (with only one list) in v2rayGeoIPDat format
$ ./geoip lookup -f v2rayGeoIPDat -u ./cn.dat 1.0.1.1
true


# 从 v2rayGeoIPDat 格式的本地文件（包含多个类别）中查找某个 IP 地址
# lookup IP from local file (with multiple lists) in v2rayGeoIPDat format
$ ./geoip lookup -f v2rayGeoIPDat -u ./geoip.dat 1.0.1.1
cn


# 从 v2rayGeoIPDat 格式的本地文件（包含多个类别）中查找某个 IP 地址
# lookup IP from local file (with multiple lists) in v2rayGeoIPDat format
$ ./geoip lookup -f v2rayGeoIPDat -u ./geoip.dat 1.0.0.1
au,cloudflare


# 从 v2rayGeoIPDat 格式的远程 URL（包含多个类别）中查找某个 CIDR
# lookup CIDR from remote URL (with multiple lists) in v2rayGeoIPDat format
$ ./geoip lookup -f v2rayGeoIPDat -u https://example.com/geoip.dat 1.0.0.1/24
au,cloudflare




# ================= REPL Mode ================= #

# 从 text 格式的本地文件（只包含一个类别）中查找某个 IP 地址或 CIDR
# lookup IP or CIDR from local file (with only one list) in text format
$ ./geoip lookup -f text -u ./cn.txt
Enter IP or CIDR (type "exit" to quit):
>> 1.0.1.1
true

>> 1.0.1.1/24
true

>> 1.0.1.1/23
false

>> 2.2.2.2
false

>> 2.2.2.2/24
false

>> 300.300.300.300
false

>> 300.300.300.300/24
false

>> exit


# 从 text 格式的远程 URL（只包含一个类别）中查找某个 IP 地址或 CIDR
# lookup IP or CIDR from remote URL (with only one list) in text format
$ ./geoip lookup -f text -u https://example.com/cn.txt
Enter IP or CIDR (type "exit" to quit):
>> 1.0.1.1
true

>> 1.0.1.1/24
true

>> 1.0.1.1/23
false

>> 2.2.2.2
false

>> 2.2.2.2/24
false

>> 300.300.300.300
false

>> 300.300.300.300/24
false

>> exit


# 从 v2rayGeoIPDat 格式的本地文件（只包含一个类别）中查找某个 IP 地址或 CIDR
# lookup IP or CIDR from local file (with only one list) in v2rayGeoIPDat format
$ ./geoip lookup -f v2rayGeoIPDat -u ./cn.dat
Enter IP or CIDR (type "exit" to quit):
>> 1.0.1.1
true

>> 1.0.1.1/24
true

>> 1.0.1.1/23
false

>> 2.2.2.2
false

>> 2.2.2.2/24
false

>> 300.300.300.300
false

>> 300.300.300.300/24
false

>> exit


# 从 v2rayGeoIPDat 格式的远程 URL（包含多个类别）中查找某个 IP 地址或 CIDR
# lookup IP or CIDR from remote URL (with multiple list) in v2rayGeoIPDat format
$ ./geoip lookup -f v2rayGeoIPDat -u https://example.com/geoip.dat
Enter IP or CIDR (type "exit" to quit):
>> 1.0.1.1
cn

>> 1.0.1.1/24
cn

>> 1.0.1.1/23
false

>> 1.0.0.1
au,cloudflare

>> 1.0.0.1/24
au,cloudflare

>> 300.300.300.300
false

>> 300.300.300.300/24
false

>> exit
```

## 使用本项目的项目

- [@Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat)
- [@Loyalsoldier/clash-rules](https://github.com/Loyalsoldier/clash-rules)
- [@Loyalsoldier/surge-rules](https://github.com/Loyalsoldier/surge-rules)

## License

[CC-BY-SA-4.0](https://creativecommons.org/licenses/by-sa/4.0/) and [GPL-3.0](https://github.com/Loyalsoldier/geoip/blob/master/LICENSE-GPL)

This product includes GeoLite2 data created by MaxMind, available from [MaxMind](https://www.maxmind.com).

## 项目 Star 数增长趋势

[![Stargazers over time](https://starchart.cc/Loyalsoldier/geoip.svg)](https://starchart.cc/Loyalsoldier/geoip)
