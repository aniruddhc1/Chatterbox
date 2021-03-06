#!/bin/bash
#testPaxos

python purge.py
go install github.com/cmu440/chatterbox/
./bin/chatterbox -isMaster=true -N=5 -port=8080 -registerAll=false &
PID1=$!
./bin/chatterbox -isMaster=false -N=5 -port=9990 -registerAll=false &
PID2=$!
./bin/chatterbox -isMaster=false -N=5 -port=8081 -registerAll=false &
PID3=$!
./bin/chatterbox -isMaster=false -N=5 -port=8082 -registerAll=false &
PID4=$!
./bin/chatterbox -isMaster=false -N=5 -port=8083 -registerAll=false &
PID5=$!
./bin/chatterbox -isMaster=false -N=5 -port=999 -registerAll=true -testNum=0

kill -9 $PID1
kill -9 $PID2
kill -9 $PID3
kill -9 $PID4
kill -9 $PID5