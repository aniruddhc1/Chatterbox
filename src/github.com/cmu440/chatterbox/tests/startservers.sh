#!/bin/bash
##!/bin/bash
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


./bin/chatterbox -isMaster=true -N=2 -port=8080 -registerAll=false


./bin/chatterbox -isMaster=false -N=2 -port=9990 -registerAll=false


./bin/chatterbox -isMaster=false -N=2 -port=999 -registerAll=true

