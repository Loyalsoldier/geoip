#!/bin/bash

set -e

curl -L -O http://geolite.maxmind.com/download/geoip/database/GeoLite2-Country-CSV.zip
unzip GeoLite2-Country-CSV.zip
rm GeoLite2-Country-CSV.zip
mv GeoLite2* geoip

go get -v -t -d github.com/v2ray/geoip/...
go run $GOPATH/src/github.com/v2ray/geoip/main.go -- --country=./geoip/GeoLite2-Country-Locations-en.csv --ipv4=./geoip/GeoLite2-Country-Blocks-IPv4.csv --ipv6=./geoip/GeoLite2-Country-Blocks-IPv6.csv

mkdir ./publish
mv ./geoip.dat ./publish/
