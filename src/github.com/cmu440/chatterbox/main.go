package main

import (
	"github.com/cmu440/chatterbox/paxos"
	"fmt"
	"errors"
	"flag"
	"time"
	"github.com/cmu440/chatterbox/chatclient"
	"encoding/json")

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
func TestGetServers(cclient *chatclient.ChatClient) error{
	args := paxos.GetServersArgs{}
	reply := paxos.GetServersReply{}

	errCall := cclient.ClientConn.Call("PaxosServer.GetServers", &args, &reply)

	if errCall != nil {
		fmt.Println("Couldln't call get servers", errCall)
		return errCall
	}

	if(!reply.Ready){
		return errors.New("servers weren't ready...?")
	}else if(len(reply.Servers) != 2){
		return errors.New("number of servers not 2")
	}
	fmt.Println("PASS GetServers()")
	return nil
}


//													//
//					TEST ACCEPTOR					//
//													//

/*
 * Check that after the proposer sends a propose request the acceptor replies OK
 * if it hasn't seen anything before.
 */
func testBasic1() error{
	msg := chatclient.ChatMessage{"Soumya", "testRoom", "TESTING", time.Now()}

	bytes, marshallErr := json.Marshal(msg)

	if marshallErr != nil {
		return errors.New("Couldn't Marshal chat message")
	}

	args := paxos.SendMessageArgs{bytes}

	errPropose := ps1.ProposeRequest(&args, &paxos.DefaultReply{})

	if errPropose != nil {
		return errPropose
	}

	return nil
}

func main(){
	isMaster := flag.Bool("isMaster", false, "to check if its a master server or not ")
	registerAll := flag.Bool("registerAll", false, "start test cases once all servers have been registered")

	N := flag.Int("N", 0, "to specify the number of servers")
	port := flag.Int("port", 1111, "to specify the port number to start the master server on")
	flag.Parse()

	fmt.Println("IN MAIN", *isMaster)

	if *registerAll {
		//CALL ALL TESTS
		cClient, _ := chatclient.NewChatClient("localhost:2000")
		err := TestGetServers(cClient)
		fmt.Println(err)
	} else if (*isMaster){
		//START THE MASTER SERVER
		ps1, err = paxos.NewPaxosServer("", *N, *port) //starting master server
		if(err != nil){
			fmt.Println(err)
		}
		select{}
	}else{
		//START ALL OTHER SERVERS
		ps2, err2 = paxos.NewPaxosServer("localhost:8080", *N, *port) // starting node

		if(err2 != nil){
			fmt.Println(err2)
		}
		select{}
	}
}
