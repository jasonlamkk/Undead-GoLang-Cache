#!/bin/bash

#

export GOPATH="`pwd`/gopath/";
echo $GOPATH;
go get github.com/gorilla/websocket
go get github.com/satori/go.uuid
go get github.com/kataras/iris/...
cd gopath/src/jason/server
echo "`pwd`"
go build jason/server
chmod +x ./server
./server -c "$GOPATH../configs/dev.toml" & 
sleep 1s
./server -c "$GOPATH../configs/dev2.toml" &
sleep 1s
./server -c "$GOPATH../configs/dev3.toml" &

echo "\r\n-----will start testing------\r\n"
sleep 2s

token=$(curl -H "Content-Type: application/json" -X POST -d '[["22.3964","114.1095"],["22.3964","114.1095"],["22.3964","114.1095"]]' http://192.168.1.194:5000/route | python -c "import json,sys;obj=json.load(sys.stdin);print obj['token'];")

echo "\r\n-----will get route from------\r\n"
echo "http://192.168.1.194:5200/route/$token"

echo "\r\n-----result------\r\n"
curl "http://192.168.1.194:5200/route/$token"
echo "\r\n-----end------\r\n"
pkill -f ./server
