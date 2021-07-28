# 简介

本项目每周四自动生成 `geoip.dat` 路由规则文件，适用于 [V2Ray](https://github.com/v2fly/v2ray-core)、[Xray-core](https://github.com/XTLS/Xray-core)、[Trojan-Go](https://github.com/p4gefau1t/trojan-go)。

## 与官方版 geoip.dat 不同之处

- 中国大陆 IPv4 地址数据使用 [IPIP.net](https://github.com/17mon/china_ip_list/blob/master/china_ip_list.txt)
- 新增类别（方便有特殊需求的用户使用）：
  - `geoip:cloudflare`
  - `geoip:cloudfront`
  - `geoip:facebook`
  - `geoip:fastly`
  - `geoip:netflix`
  - `geoip:telegram`

## 说明

[Fork](https://github.com/Loyalsoldier/geoip/fork) 本项目后，需要在自己仓库的 **[Settings]** 选项卡的 **[Secrets]** 页面中添加一个名为 **MAXMIND_GEOLITE2_LICENSE** 的 secret，否则 GitHub Actions 会运行失败。这个 secret 的值为 MAXMIND 账号的 LICENSE KEY，需要[**注册 MAXMIND 账号**](https://www.maxmind.com/en/geolite2/signup)后，在[**个人账号管理页面**](https://www.maxmind.com/en/account)左侧边栏的 **[Services]** 项下的 **[My License Key]** 里生成。

[Fork](https://github.com/Loyalsoldier/geoip/fork) 本项目后，在自己仓库的 `data` 目录中添加包含 CIDR 列表的文件，即可往 `geoip.dat` 规则文件内新增类别，文件名即为类别名。如果在 `data` 目录中添加名为 `us` 的文件，即可覆盖掉原本美国的 IPv4 和 IPv6 地址。

## 下载地址

> 如果无法访问域名 `raw.githubusercontent.com`，可以使用第二个地址（`cdn.jsdelivr.net`），但是内容更新会有 12 小时的延迟。

- **geoip.dat**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat)
- **geoip.dat.sha256sum**：
  - [https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat.sha256sum](https://raw.githubusercontent.com/Loyalsoldier/geoip/release/geoip.dat.sha256sum)
  - [https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat.sha256sum](https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/geoip.dat.sha256sum)

## 参考配置

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
        "8.8.4.4/32",
        "geoip:telegram"
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

## License

[CC-BY-SA-4.0](https://creativecommons.org/licenses/by-sa/4.0/)

This product includes GeoLite2 data created by MaxMind, available from [MaxMind](http://www.maxmind.com).
