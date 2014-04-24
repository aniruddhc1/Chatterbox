#!/bin/sh

go install github.com/cmu440/chatterbox/
./bin/chatterbox -isMaster=true -N=3 -port=8080 -registerAll=false &
#pid1 = $! //figure this out later
./bin/chatterbox -isMaster=false -N=3 -port=9990 -registerAll=false &
./bin/chatterbox -isMaster=false -N=3 -port=9991 -registerAll=false &
./bin/chatterbox -isMaster=false -N=3 -port=420 -registerAll=true &

