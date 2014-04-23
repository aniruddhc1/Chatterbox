package main

import (
	"fmt"
	"errors"
	"flag"
	"time"
	"github.com/cmu440/chatterbox/chatclient"
	"encoding/json"
	"github.com/cmu440/chatterbox/multipaxos"
)


// 												//
//					INIT SERVERS 			 	//
//												//
func StartServer1(){
	_, err := multipaxos.NewPaxosServer("", 1, 8080) //starting master server
	if(err != nil){
		fmt.Println(err)
	}

}

func StartServer2(){
	_, err2 := multipaxos.NewPaxosServer("localhost:8080", 2, 9999) // starting node

	if(err2 != nil){
		fmt.Println(err2)
	}
}


/*
 * testing if GetServers returns the correct number of the servers once all have joined
 */
func TestGetServers(cclient *chatclient.ChatClient) error{
	args := &multipaxos.GetServersArgs{}
	reply := &multipaxos.GetServersReply{}

	if cclient == nil {
		fmt.Println("HEREEE")
	}

	errCall := cclient.GetServers(args, reply)

	if errCall != nil {
		fmt.Println("Couldln't call get servers", errCall)
		return errCall
	}

	//													//
	//					TEST GENERAL 					//
	//													//

	if(!reply.Ready){
		return errors.New("servers weren't ready...?")
	}else if(len(reply.Servers) != 2){
		return errors.New("number of servers not 2")
	}
	fmt.Println("PASS GetServers()")
	return nil
}


//													//
//					TEST PROPOSER 					//
//													//



//													//
//					TEST ACCEPTOR					//
//													//

/*
 * Check that after the proposer sends a propose request the acceptor replies OK
 * if it hasn't seen anything before.
 */
func testBasic1(cclient *chatclient.ChatClient) error{
	fmt.Println("starting testbasic1")
	msg := chatclient.ChatMessage{"Soumya", "testRoom", "TESTING", time.Now()}

	bytes, marshalErr := json.Marshal(msg)

	if marshalErr != nil {
		return errors.New("Couldn't Marshal chat message")
	}

	args := &multipaxos.SendMessageArgs{bytes,
									multipaxos.Tester{
										KillStage : "",
										KillTime : "",
									},}

	err := cclient.SendMessage(args, &multipaxos.SendMessageReplyArgs{})


	if err != nil {
		return err
	}
	return nil
}

func testBasic2(cClient1, cClient2 *chatclient.ChatClient) error {
	fmt.Println("starting testbasic2")

	msg1 := chatclient.ChatMessage{"Aniruddh", "testRoom", "lololollol :D", time.Now()}
	bytes1, marshalErr := json.Marshal(msg1)
	if(marshalErr != nil){
		return errors.New("error occurred while marshaling msg1 in testBasic2")
	}

	msg2 := chatclient.ChatMessage{"Soumya", "testRoom", "get out hahaa", time.Now()}
	bytes2, marshalErr1 := json.Marshal(msg2)
	if(marshalErr1 != nil){
		return errors.New("error occurred while marshaling msg2 in testBasic2")
	}

	args1 := &multipaxos.SendMessageArgs{Value : bytes1,
										Tester : multipaxos.Tester{
											KillStage : "sendAccept",
											KillTime : "end",
										},}
	args2 := &multipaxos.SendMessageArgs{Value : bytes2,
										Tester : multipaxos.Tester{
											KillStage : "",
											KillTime : "",
										},}

	/*
	killStage : "sendPropose", "sendAccept", "sendCommit"
				"receivePropose", "receiveAccept", //todo
				"receiveCommit" //todo
	killTime : start, mid, end
	*/

	err := cClient1.SendMessage(args1, &multipaxos.SendMessageReplyArgs{})

	if(err != nil){
		return err
	}

	err2 := cClient2.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})
	if(err2 != nil){
		return err2
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
		var cClient *chatclient.ChatClient
//		var cClient1 *chatclient.ChatClient
		cClient, _ = chatclient.NewChatClient("localhost:2000", 8080)
//		cClient1, _ = chatclient.NewChatClient("localhost:3000", 9990)
//		err := TestGetServers(cClient)
//		fmt.Println(err)
		err1 := testBasic1(cClient)
		fmt.Println(err1)
//		err2 := testBasic2(cClient, cClient1)
//		fmt.Println(err2)
	} else if (*isMaster){
		//START THE MASTER SERVER
		_, err := multipaxos.NewPaxosServer("", *N, *port) //starting master server
		if(err != nil){
			fmt.Println(err)
		}
		select{}
	}else{
		//START ALL OTHER SERVERS
		_, err2 := multipaxos.NewPaxosServer("localhost:8080", *N, *port) // starting node

		if(err2 != nil){
			fmt.Println(err2)
		}
		select{}
	}
}
