#!/bin/bash

set -e

apt -y install curl unzip git jq

mkdir /v2
cd /v2

echo "Installing Go runtime"
GO_INSTALL=golang.tar.gz
curl -L -o ${GO_INSTALL} https://storage.googleapis.com/golang/go1.11.2.linux-amd64.tar.gz
tar -C /usr/local -xzf ${GO_INSTALL}
export GOPATH=/v2/go
export PATH=$PATH:${GOPATH}/bin:/usr/local/go/bin

function getattr() {
  curl -s -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/$2/attributes/$1
}

GITHUB_TOKEN=$(getattr "github_token" "project")

RELEASE_TAG=$(date +%Y%m%d)
echo "Releasing GeoIP at ${RELEASE_TAG}"

curl -L -O http://geolite.maxmind.com/download/geoip/database/GeoLite2-Country-CSV.zip
unzip GeoLite2-Country-CSV.zip
rm GeoLite2-Country-CSV.zip
mv GeoLite2* geoip

CGO_ENABLED=0 go get -u v2ray.com/ext/tools/geoip/main
main --country=./geoip/GeoLite2-Country-Locations-en.csv --ipv4=./geoip/GeoLite2-Country-Blocks-IPv4.csv --ipv6=./geoip/GeoLite2-Country-Blocks-IPv6.csv

JSON_DATA=$(echo "{}" | jq -c ".tag_name=\"${RELEASE_TAG}\"")
RELEASE_ID=$(curl --data "${JSON_DATA}" -H "Authorization: token ${GITHUB_TOKEN}" -X POST https://api.github.com/repos/v2ray/geoip/releases | jq ".id")

function uploadfile() {
  FILE=$1
  CTYPE=$(file -b --mime-type $FILE)
  curl -H "Authorization: token ${GITHUB_TOKEN}" -H "Content-Type: ${CTYPE}" --data-binary @$FILE "https://uploads.github.com/repos/v2ray/geoip/releases/${RELEASE_ID}/assets?name=$(basename $FILE)"
}

uploadfile ./geoip.dat
