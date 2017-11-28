#!/bin/bash


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
# ./server_kill_first -c "$GOPATH../configs/dev.toml" & 
# sleep 1s
# ./server -c "$GOPATH../configs/dev2.toml" &
# sleep 1s
# ./server -c "$GOPATH../configs/dev3.toml" &


# token=$(curl -H "Content-Type: application/json" -X POST -d '[["22.2908569","114.1988573"],["22.2896519","114.1944555"],["22.2801243","114.1837031"],["22.2773764","114.1799277"],["22.2866291","114.1928005"],["22.2835264","114.1924385"]]' http://192.168.1.194:5000/route | python -c "import json,sys;obj=json.load(sys.stdin);print obj['token'];")

# echo "\r\n-----expect result is in progress------\r\n"
# echo ":5000/route/$token"
# curl "http://192.168.1.194:5000/route/$token"