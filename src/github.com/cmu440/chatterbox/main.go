package main

import (
	"github.com/cmu440/chatterbox/paxos"
	"fmt"
)

func main(){
	_, err := paxos.NewPaxosServer("", 2, 8080) //starting master server
	//_, err1 := paxos.NewPaxosServer("localhost:8080", 2, 9999) // starting node
	fmt.Println(err)
	//fmt.Println(err1)

}
