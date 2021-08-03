#!/usr/bin/env bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH

GEOIP_URL="https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/geoip.dat"
GEOSITE_URL="https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/geosite.dat"

data_folder="/usr/local/share/xray/"
proxy_address="http://127.0.0.1:8888"
status_code='0'

if [[ "$UID" -ne '0' ]];
then echo "You must be root to run this!"
      exit 1
fi

cd $data_folder || exit 1
#rm -f geoip.dat.bak
#rm -f geosite.dat.bak

mv geoip.dat geoip.dat.bak
mv geosite.dat geosite.dat.bak

#download geoip
if ! curl -x $proxy_address -O $GEOIP_URL --silent;then
	mv geoip.dat.bak geoip.dat
	echo "faild to download geoip.dat" >> /home/root/update.log
	status_code=1
fi

#download geosite
if ! curl -x $proxy_address -O $GEOSITE_URL --silent;then
	mv geosite.dat.bak geosite.dat
	echo "faild to download geosite.dat" >> /home/root/update.log
	status_code=1
fi

#restart xray
if [[ "$status_code" -eq '0' ]];then
	systemctl restart xray
else
	exit 2
fi

echo -e "$(date)\n" >> /home/root/update.log
