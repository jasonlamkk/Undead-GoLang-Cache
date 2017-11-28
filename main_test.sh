#!/bin/bash

#

export GOPATH="`pwd`/gopath/";
echo $GOPATH;
go get googlemaps.github.io/maps
go get github.com/gorilla/websocket
go get github.com/satori/go.uuid
go get github.com/kataras/iris/...
cd gopath/src/jason/server
echo "`pwd`"
go build jason/server
cp ./server ./server_kill_first
chmod +x ./server
chmod +x ./server_kill_first

./server_kill_first -c "$GOPATH../configs/dev.toml" & 
sleep 1s
./server -c "$GOPATH../configs/dev2.toml" &
sleep 1s
./server -c "$GOPATH../configs/dev3.toml" &

echo "\r\n-----will start testing------\r\n"
sleep 2s

token=$(curl -H "Content-Type: application/json" -X POST -d '[["22.2908569","114.1988573"],["22.2896519","114.1944555"],["22.2801243","114.1837031"],["22.2773764","114.1799277"],["22.2866291","114.1928005"],["22.2835264","114.1924385"]]' http://192.168.1.194:5000/route | python -c "import json,sys;obj=json.load(sys.stdin);print obj['token'];")

echo "\r\n-----expect result is in progress------\r\n"
echo ":5000/route/$token"
curl -vs "http://192.168.1.194:5000/route/$token"
sleep 2s
echo "\r\n-----expect result from google map API------\r\n"
echo ":5000/route/$token"
curl -vs "http://192.168.1.194:5000/route/$token"

echo "\r\n-----start test cluster------"
echo "\r\n-----will get route from------\r\n"
echo "http://192.168.1.194:5200/route/$token"
echo "\r\n-----result------\r\n"
curl -vs "http://192.168.1.194:5200/route/$token"
echo "\r\n-----end------\r\n"


echo "\r\nstop 1st server"
pkill -f ./server_kill_first

echo "\r\n-----start survial nodes------"
echo "\r\n-----will get route from------\r\n"
echo "http://192.168.1.194:5100/route/$token"
echo "\r\n-----result------\r\n"
curl -vs "http://192.168.1.194:5100/route/$token"

echo "\r\n-----will get route from------\r\n"
echo "http://192.168.1.194:5200/route/$token"
echo "\r\n-----result------\r\n"
curl -vs "http://192.168.1.194:5200/route/$token"
echo "result from survial nodes shall output as expected"
sleep 1s
echo "\r\nstop servers"
pkill -f ./server
