package main

import "fmt"

import "github.com/cmu440/chatterbox/paxos"

import (
)

var ps1 *paxos.PaxosServer
var ps2 *paxos.PaxosServer
var err error
var err2 error

// 												//
//					INIT SERVERS 			 	//
//												//
func startServer1(){
	ps1, err = paxos.NewPaxosServer("", 1, 8080) //starting master server
	if(err != nil){
		fmt.Println(err)
	}

}

func startServer2(){
	ps2, err2 = paxos.NewPaxosServer("localhost:8080", 2, 9999) // starting node

	if(err2 != nil){
		fmt.Println(err2)
	}
}


//													//
//					TEST PROPOSER 					//
//													//




//													//
//					TEST ACCEPTOR					//
//													//

func main(){
	go startServer1()
	go startServer2()

}
