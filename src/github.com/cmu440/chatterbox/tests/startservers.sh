#!/bin/bash
#
#go install github.com/cmu440/chatterbox/
#
#PAXOS_SERVER=$GOPATH/bin/chatterbox
#
#
#function testPaxos {
#    ${PAXOS_SERVER} -isMaster=${isMaster} -N=${N} -port=${STORAGE_PORT_MASTER} -registerAll=${registerAll}
#    STORAGE_SERVER_PID[0]=$!
#}
#
#testPaxos


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
./bin/chatterbox -isMaster=false -N=5 -port=999 -registerAll=true

kill -9 $PID1
kill -9 $PID2
kill -9 $PID3
kill -9 $PID4
kill -9 $PID5