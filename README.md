# GeoIP List for V2Ray

Automatically weekly release of geoip.dat for V2Ray.

This product includes GeoLite2 data created by MaxMind, available from [MaxMind](http://www.maxmind.com/), with replaced CN IPv4 CIDR available from [IPIP.net China IP List](https://github.com/17mon/china_ip_list/blob/master/china_ip_list.txt).

## Download links

- **geoip.dat**：[https://github.com/Loyalsoldier/geoip/raw/release/geoip.dat](https://github.com/Loyalsoldier/geoip/raw/release/geoip.dat)
- **geoip.dat.sha256sum**：[https://github.com/Loyalsoldier/geoip/raw/release/geoip.dat.sha256sum](https://github.com/Loyalsoldier/geoip/raw/release/geoip.dat.sha256sum)

## Usage example

```json
"routing": {
  "rules": [
    {
      "type": "field",
      "outboundTag": "Direct",
      "ip": [
        "223.5.5.5/32",
        "119.29.29.29/32",
        "180.76.76.76/32",
        "114.114.114.114/32",
        "geoip:cn",
        "geoip:private"
      ]
    },
    {
      "type": "field",
      "outboundTag": "Proxy-1",
      "ip": [
        "1.1.1.1/32",
        "1.0.0.1/32",
        "8.8.8.8/32",
        "8.8.4.4/32"
      ]
    },
    {
      "type": "field",
      "outboundTag": "Proxy-2",
      "ip": [
        "geoip:us",
        "geoip:ca"
      ]
    },
    {
      "type": "field",
      "outboundTag": "Proxy-3",
      "ip": [
        "geoip:hk",
        "geoip:tw",
        "geoip:jp",
        "geoip:sg"
      ]
    }
  ]
}
```
