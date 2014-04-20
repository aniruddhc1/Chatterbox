#!/bin/bash

go install github.com/cmu440/chatterbox/

PAXOS_SERVER=$GOPATH/bin/chatterbox

let STORAGE_PORT_MASTER=8080
let STORAGE_PORT_SLAVE=9000
let N=2

function startPaxosServers {
    ${PAXOS_SERVER} -isMaster=true -N=${N} -port=${STORAGE_PORT_MASTER}
    STORAGE_SERVER_PID[0]=$!

    ${PAXOS_SERVER} -isMaster=false -N=${N} -port=${STORAGE_PORT_SLAVE}
    STORAGE_SERVER_PID[1]=$!
}

function testPaxos {
    startPaxosServers

}

testPaxos