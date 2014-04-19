package main

import (
	"github.com/cmu440/chatterbox/paxos"
	"fmt"
	"errors"
	"flag"
	"time"
)

var ps1 *paxos.PaxosServer
var ps2 *paxos.PaxosServer
var err error
var err2 error

// 												//
//					INIT SERVERS 			 	//
//												//
func StartServer1(){
	ps1, err = paxos.NewPaxosServer("", 1, 8080) //starting master server
	if(err != nil){
		fmt.Println(err)
	}

}

func StartServer2(){
	ps2, err2 = paxos.NewPaxosServer("localhost:8080", 2, 9999) // starting node

	if(err2 != nil){
		fmt.Println(err2)
	}
}

//													//
//					TEST PROPOSER 					//
//													//

/*
 * testing if GetServers returns the correct number of the servers once all have joined
 */
func TestGetServers() error{
	args := paxos.GetServersArgs{}
	reply := paxos.GetServersReply{}

	if ps2 == nil {
		fmt.Println("server is nil")
	}


	ps2.GetServers(&args, &reply)

	if(!reply.Ready){
		return errors.New("servers weren't ready...?")
	}else if(len(reply.Servers) != 2){
		return errors.New("number of servers not 2")
	}
	return nil
}


//													//
//					TEST ACCEPTOR					//
//													//

func main(){
	time.Sleep(time.Second * 3)
//	err := TestGetServers()

	isMaster := flag.Bool("isMaster", false, "to check if its a master server or not ")

	flag.Parse()

	fmt.Println("IN MAIN", *isMaster)
	if(*isMaster){
		StartServer1()
	}else{
		StartServer2()
	}
}
