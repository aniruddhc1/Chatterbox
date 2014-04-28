#!/bin/bash

go install github.com/cmu440/chatterbox/
./bin/chatterbox -isMaster=true -N=5 -port=8080 -startChat=false &
PID1=$!
./bin/chatterbox -isMaster=false -N=5 -port=9990 -startChat=false &
PID2=$!
./bin/chatterbox -isMaster=false -N=5 -port=8081 -startChat=false &
PID3=$!
./bin/chatterbox -isMaster=false -N=5 -port=8082 -startChat=false &
PID4=$!
./bin/chatterbox -isMaster=false -N=5 -port=8083 -startChat=false &
PID5=$!
./bin/chatterbox -isMaster=false -N=5 -port=1050 -startChat=true

kill -9 $PID1
kill -9 $PID2
kill -9 $PID3
kill -9 $PID4
kill -9 $PID5

