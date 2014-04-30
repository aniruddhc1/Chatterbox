package main

import (
	"fmt"
	"errors"
	"flag"
	"time"
	"github.com/cmu440/chatterbox/chatclient"
	"encoding/json"
	"github.com/cmu440/chatterbox/multipaxos"
	"bytes"
)

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

	if(!reply.Ready){
		return errors.New("servers weren't ready...?")
	}else if(len(reply.Servers) != 5){
		return errors.New("number of servers not 2")
	}
	fmt.Println("PASS GetServers()")
	return nil
}

/*
 * Check that after the proposer sends a propose request the acceptor replies OK
 * if it hasn't seen anything before. Does one iteration of paxos without any
 * failure of nodes
 */
func testBasic1(cclient *chatclient.ChatClient) error{
	fmt.Println("starting testbasic1")
	msg := chatclient.ChatMessage{"Soumya", "testRoom", "TESTING", time.Now(), ""}

	bytes, marshalErr := json.Marshal(msg)

	if marshalErr != nil {
		return errors.New("Couldn't Marshal chat message")
	}

	args := &multipaxos.SendMessageArgs{bytes,
									multipaxos.Tester{
										Stage : "",
										Time : "",
										Kill : true,
										SleepTime : 0,
									},
									8080}

	err := cclient.SendMessage(args, &multipaxos.SendMessageReplyArgs{})


	if err != nil {
		return err
	}
	fmt.Println("pass test basic 1")
	return nil
}



/*
 * Does one iteration of paxos with the failure of first proposer after sending
 * accept messages. Test that it commits the right message (the first one should be
 * commited)
 */
func testBasic2(cClient1 *chatclient.ChatClient, port1 int, port2 int) error {
	fmt.Println("starting testbasic2")

	msg1 := chatclient.ChatMessage{"Aniruddh", "testRoom", "lololollol :D", time.Now(), ""}
	bytes1, marshalErr := json.Marshal(msg1)
	if(marshalErr != nil){
		return errors.New("error occurred while marshaling msg1 in testBasic2")
	}

	msg2 := chatclient.ChatMessage{"Soumya", "testRoom", "get out hahaa", time.Now(), ""}
	bytes2, marshalErr1 := json.Marshal(msg2)
	if(marshalErr1 != nil){
		return errors.New("error occurred while marshaling msg2 in testBasic2")
	}

	args1 := &multipaxos.SendMessageArgs{Value : bytes1,
										Tester : multipaxos.Tester{
											Stage : "sendAccept",
											Time : "end",
											Kill : true,
											SleepTime : 0,
										},
										PaxosPort : port1}

	args2 := &multipaxos.SendMessageArgs{Value : bytes2,
										Tester : multipaxos.Tester{
											Stage : "",
											Time : "",
											Kill : true,
											SleepTime : 0,
										},
										PaxosPort : port2}


	//send the first message, the proposer will die before it sends a commit message
	err := cClient1.SendMessage(args1, &multipaxos.SendMessageReplyArgs{})

	if(err != nil){
		return err
	}

	time.Sleep(time.Second)

	fileArgs1 := &multipaxos.FileArgs{Port : port1,}
	fileReply1 := &multipaxos.FileReply{}
	errLog1 := cClient1.GetLogFile(fileArgs1, fileReply1)

	fileArgs2 := &multipaxos.FileArgs{Port : port2,}
	fileReply2 := &multipaxos.FileReply{}
	errLog2 := cClient1.GetLogFile(fileArgs2, fileReply2)

	if(errLog2 != nil || errLog1 != nil){
		fmt.Println(errLog1, errLog2)
		return errors.New("error occurred while getting the log file")
	}

	if len(fileReply1.File) != 0 || len(fileReply2.File) != 0 {
		return errors.New("Something got commited even though the proposer failed to send commit messages")
	}

	//Sending a second message from a different proposer
	error2 := cClient1.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})
	if(error2 != nil){
		return error2
	}

	time.Sleep(time.Second*2)

	//Check logs again
	fileReplyFailedProposer := &multipaxos.FileReply{}
	fileReplyThisProposer := &multipaxos.FileReply{}
	errLog1 = cClient1.GetLogFile(fileArgs1, fileReplyFailedProposer)
	errLog2 = cClient1.GetLogFile(fileArgs2, fileReplyThisProposer)

	if(errLog2 != nil || errLog1 != nil){
		fmt.Println(errLog1, errLog2)
		return errors.New("error occurred while getting the log file")
	}

	if len(fileReplyFailedProposer.File) != 0 {
		return errors.New("The failed proposer should not have commited anything")
	}

	fmt.Println(fileReplyThisProposer.File)
	if len(fileReplyThisProposer.File) == 0{
		return errors.New("The second proposer should've been able to commit a message")
		//Check that the right message has been commited
		message := &chatclient.ChatMessage{}
		err := json.Unmarshal(fileReplyThisProposer.File, message)
		if err != nil {
			return errors.New("Couldn't get message from log file")
		}

		if *message != msg1 {
			return errors.New("The message commited should be the first message")
		}
	}

	fmt.Println("Pass test basic 2")

	return nil
}


/*
 * tests shannon's scenario from her ppt
 */
func testBasic3(cClient1 *chatclient.ChatClient, port1 int, port2 int, port3 int) error {
	msg1 := chatclient.ChatMessage{"Srini", "testRoom", "lololollol :D", time.Now(), ""}
	bytes1, marshalErr := json.Marshal(msg1)
	if (marshalErr != nil) {
		return errors.New("error occurred while marshaling msg1 in testBasic2")
	}

	msg2 := chatclient.ChatMessage{"Taran", "testRoom", "get out hahaa", time.Now(), ""}
	bytes2, marshalErr1 := json.Marshal(msg2)
	if (marshalErr1 != nil) {
		return errors.New("error occurred while marshaling msg2 in testBasic2")
	}


	msg3 := chatclient.ChatMessage{"Sally", "testRoom", "cool stories", time.Now(), ""}
	bytes3, marshalErr2 := json.Marshal(msg3)
	if (marshalErr2 != nil) {
		return errors.New("error occurred while marshaling msg3 in testBasic2")
	}

	args1 := &multipaxos.SendMessageArgs{Value : bytes1,
		Tester : multipaxos.Tester{
			Stage : "sendPropose",
			Time : "start",
			Kill : false,
			SleepTime : 30,
		},
		PaxosPort : port1}

	args2 := &multipaxos.SendMessageArgs{Value : bytes2,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2}


	args3 := &multipaxos.SendMessageArgs{Value : bytes3,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort: port3}

	cClient1.SendMessage(args1, &multipaxos.SendMessageReplyArgs{})

	time.Sleep(time.Second)
	fmt.Println("SENDING SECOND MESSAGEEEEEE")
	err2 := cClient1.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})
	if (err2 != nil) {
		fmt.Println("Error 2", err2)
		return err2
	}

	time.Sleep(time.Second * 20)

	err3 := cClient1.SendMessage(args3, &multipaxos.SendMessageReplyArgs{})
	if (err3 != nil) {
		fmt.Println("Error 3", err3)
		return err3
	}

	time.Sleep(time.Second * 2)
	//TODO send the first one make and it wait right before it sends accept requests
	//then do send args 2 while its waiting
	//then wait for like 3 seconds nd try
	//sending args 3

	fmt.Println("pass test basic 3")
	return nil
}

/**
 * Simulates the case of one node dying "test a dead node test" on piazza
 */
func testBasic4(cclient *chatclient.ChatClient, port1, port2, port3, port4, port5 int) error {
	msg1 := chatclient.ChatMessage{"alockwoo", "awesome", "poopmaster", time.Now(), ""}
	msg2 := chatclient.ChatMessage{"achaturv", "awesome", "golang ", time.Now(), ""}
	msg3 := chatclient.ChatMessage{"skethu", "awesome", "i'm too cool", time.Now(), ""}
	msg4 := chatclient.ChatMessage{"cgarrod", "awesome", "i love webapps!", time.Now(), ""}

	bytes1, err1 := json.Marshal(msg1)
	bytes2, err2 := json.Marshal(msg2)
	bytes3, err3 := json.Marshal(msg3)
	bytes4, err4 := json.Marshal(msg4)

	if (err1 != nil || err2 != nil || err3 != nil || err4 != nil) {
		return errors.New("error occurred while marshaling")
	}

	argsKill := &multipaxos.SendMessageArgs{
		Value : make([]byte, 1),
		Tester : multipaxos.Tester{
			Stage : "sendPropose",
			Time : "start",
			Kill : true,
			SleepTime : 0,
		},
		PaxosPort : port5,
	}

	//kill server 5 in the beginning before doing anything
	er := cclient.SendMessage(argsKill, &multipaxos.SendMessageReplyArgs{})
	if(er != nil){
		fmt.Println(er)
		return er
	}

	time.Sleep(time.Second)

	args1 := &multipaxos.SendMessageArgs{
		Value : bytes1,
		Tester : multipaxos.Tester{
			Stage : "" ,
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}

	errSend1 := cclient.SendMessage(args1, &multipaxos.SendMessageReplyArgs{})
	if(errSend1 != nil){
		fmt.Println(errSend1)
		return errSend1
	}

	time.Sleep(time.Second * 2)

	fmt.Println("checking logs now for round 1")

	//see that logs for 1,2,3,4 have a line in them

	file1 := &multipaxos.FileReply{}
	file2 := &multipaxos.FileReply{}
	file3 := &multipaxos.FileReply{}
	file4 := &multipaxos.FileReply{}
	file5 := &multipaxos.FileReply{}

	errLogR1 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1)
	errLogR2 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2)
	errLogR3 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3)
	errLogR4 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4)
	errLogR5 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5)

	if (errLogR1 != nil || errLogR2 != nil || errLogR3 != nil ||
		errLogR4 != nil || errLogR5 != nil) {
		return errors.New("error occurred while getting the log file =| ")
	}


	fmt.Println("it got the files from all the servers! ")

	//check if message been committed to 1,2,3,4 and not to 5. fails upon violation of
	//any of these
	if (bytes.Compare(file1.File, file2.File) != 0) ||
			(bytes.Compare(file2.File, file3.File) != 0) ||
			(bytes.Compare(file3.File, file4.File) != 0) ||
			(bytes.Compare(file4.File, file5.File) == 0) {
		return errors.New("inconsistency between log files in the first pass")
	}else{
		fmt.Println("passed round 1 with 1,2,3,4 alive; 5 dead")
		fmt.Println("no inconsistencies found in round 1 of testbasic4")
	}

	//delay for server 4 at the very beginning
	argsDelay := &multipaxos.SendMessageArgs{
		Value : make([]byte, 1),
		Tester : multipaxos.Tester{
			Stage : "sendPropose",
			Time : "start",
			Kill : false,
			SleepTime : 4,
		},
		PaxosPort : port4,
	}

	//delaying server4
	er1 := cclient.SendMessage(argsDelay, &multipaxos.SendMessageReplyArgs{})

	if er1 != nil{
		fmt.Println(er1)
		return er1
	}

	//check for log consistency
	if (bytes.Compare(file1.File, file2.File) != 0) ||
			(bytes.Compare(file2.File, file3.File) != 0) ||
			(bytes.Compare(file3.File, file4.File) != 0) ||
			(bytes.Compare(file4.File, file5.File) == 0) {
		return errors.New("inconsistency between log files in the first pass")
	}else{
		fmt.Println("passed round 1 with 1,2,3,4 alive; 5 dead")
		fmt.Println("no inconsistencies found in round 1 of testbasic4")
	}

	args2 := &multipaxos.SendMessageArgs{
		Value : bytes2,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port1,
	}

	time.Sleep(time.Second*1)
	errSend2 := cclient.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})
	if(errSend2 != nil){
		fmt.Println(errSend2)
		return errSend2
	}

	time.Sleep(time.Second*2)
	//compare logs, make sure that 1,2,3 are consistent

	file1R2 := &multipaxos.FileReply{}
	file2R2 := &multipaxos.FileReply{}
	file3R2 := &multipaxos.FileReply{}
	file4R2 := &multipaxos.FileReply{}
	file5R2 := &multipaxos.FileReply{}

	cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1R2)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2R2)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3R2)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4R2)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5R2)

	//1,2,3 should have same logs; 4 and 5 should have different logs (4 is delayed, 5 is killed)
	if ((!bytes.Equal(file1R2.File, file2R2.File)) ||
			(!bytes.Equal(file2R2.File, file3R2.File)) ||
			(bytes.Equal(file3R2.File, file4R2.File)) ||
			(bytes.Equal(file4R2.File, file5R2.File)) ||
			(bytes.Equal(file3R2.File, file5R2.File))) {
		return errors.New("inconsistency between log files in the second round")
	}

	fmt.Println("no inconsistency; passed round 2 with 1,2,3 alive, 4 delayed, 5 dead")


	//wait till delay ends.. then make 4 send a proposal, send message.
	//make sure that 1,2,3,4 is consistent

	time.Sleep(time.Second * 4)

	args3 := &multipaxos.SendMessageArgs{
		Value : bytes3,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}

	errSend3 := cclient.SendMessage(args3, &multipaxos.SendMessageReplyArgs{})

	if(errSend3 != nil){
		fmt.Println(errSend3)
		return errSend3
	}

	time.Sleep(time.Second*2)

	file1R3 := &multipaxos.FileReply{}
	file2R3 := &multipaxos.FileReply{}
	file3R3 := &multipaxos.FileReply{}
	file4R3 := &multipaxos.FileReply{}
	file5R3 := &multipaxos.FileReply{}

	cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1R3)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2R3)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3R3)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4R3)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5R3)

	fmt.Println("File1 Round 3", file1R3)
	fmt.Println("File2 Round 3", file2R3)
	fmt.Println("File3 Round 3", file3R3)
	fmt.Println("File4 Round 3", file4R3)
	fmt.Println("File5 Round 3", file5R3)

	//1,2,3,4 alive 5 dead; expected that 4 has recovered data from one of the other partitions
	if (bytes.Compare(file1R3.File, file2R3.File) != 0) ||
			(bytes.Compare(file2R3.File, file3R3.File) != 0) ||
			(bytes.Compare(file3R3.File, file4R3.File) != 0) ||
			(bytes.Compare(file4R3.File, file5R3.File) == 0) {
		return errors.New("inconsistency between log files in the third round")
	}

	//wake up server 5 from after being killed.
	fmt.Println("pass round3")


	getServersReply := &multipaxos.GetServersReply{}
	cclient.GetServers(&multipaxos.GetServersArgs{}, getServersReply)

	conn := chatclient.PaxosServerConnections[port5]

	conn.Call("PaxosServer.WakeupServer", &multipaxos.WakeupRequestArgs{}, &multipaxos.WakeupReplyArgs{})

	//send a message from 5

	args4 := &multipaxos.SendMessageArgs{
		Value : bytes4,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}

	//make sure that 1,2,3,4,5 logs are consistent

	errSend4 := cclient.SendMessage(args4, &multipaxos.SendMessageReplyArgs{})

	if(errSend4 != nil) {
		fmt.Println(errSend4)
		return errSend4
	}

	time.Sleep(time.Second*3)
	file1R4 := &multipaxos.FileReply{}
	file2R4 := &multipaxos.FileReply{}
	file3R4 := &multipaxos.FileReply{}
	file4R4 := &multipaxos.FileReply{}
	file5R4 := &multipaxos.FileReply{}

	cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1R4)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2R4)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3R4)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4R4)
	cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5R4)

	if (bytes.Compare(file1R4.File, file2R4.File) != 0) ||
			(bytes.Compare(file2R4.File, file3R4.File) != 0) ||
			(bytes.Compare(file3R4.File, file4R4.File) != 0) ||
			(bytes.Compare(file4R4.File, file5R4.File) != 0) {
		return errors.New("not all log files are consistent!")
	}

	fmt.Println("PASS TestBasic4()")
	return nil
}

/*
 * Does 5 iterations of paxos without failure to check consistency
 */
func testBasic5(cclient *chatclient.ChatClient, port1, port2, port3, port4, port5 int) error{
	msg1 := chatclient.ChatMessage{"alockwoo", "awesome", "poopmaster", time.Now(), ""}
	msg2 := chatclient.ChatMessage{"achaturv", "awesome", "golang ", time.Now(), ""}
	msg3 := chatclient.ChatMessage{"skethu", "awesome", "i'm too cool", time.Now(), ""}
	msg4 := chatclient.ChatMessage{"cgarrod", "awesome", "i love webapps!", time.Now(), ""}
	msg5 := chatclient.ChatMessage{"rebecca", "awesome", "~~im a troll~~!", time.Now(), ""}

	bytes1, err1 := json.Marshal(msg1)
	bytes2, err2 := json.Marshal(msg2)
	bytes3, err3 := json.Marshal(msg3)
	bytes4, err4 := json.Marshal(msg4)
	bytes5, err5 := json.Marshal(msg5)

	if (err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil) {
		return errors.New("error occurred while marshaling")
	}

	args1 := &multipaxos.SendMessageArgs{
		Value : bytes1,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port1,
	}

	args2 := &multipaxos.SendMessageArgs{
		Value : bytes2,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}

	args3 := &multipaxos.SendMessageArgs{
		Value : bytes3,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port3,
	}

	args4 := &multipaxos.SendMessageArgs{
		Value : bytes4,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port4,
	}

	args5 := &multipaxos.SendMessageArgs{
		Value : bytes5,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port5,
	}

	err1S := cclient.SendMessage(args1, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second * 3)
	err2S := cclient.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second * 3)
	err3S := cclient.SendMessage(args3, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second * 3)
	err4S := cclient.SendMessage(args4, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second * 3)
	err5S := cclient.SendMessage(args5, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second * 3)

	if (err1S != nil || err2S != nil || err3S != nil ||
		err4S != nil || err5S != nil) {
		return errors.New("error occurred while sending message")
	}

	file1 := &multipaxos.FileReply{}
	file2 := &multipaxos.FileReply{}
	file3 := &multipaxos.FileReply{}
	file4 := &multipaxos.FileReply{}
	file5 := &multipaxos.FileReply{}

	errLogR1 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1)
	errLogR2 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2)
	errLogR3 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3)
	errLogR4 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4)
	errLogR5 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5)

	if (errLogR1 != nil || errLogR2 != nil || errLogR3 != nil ||
		errLogR4 != nil || errLogR5 != nil) {
		return errors.New("error occurred while getting the log file =| ")
	}

	fmt.Println("it got the files from all the servers! ")

	//check if message been committed to 1,2,3,4 and not to 5. fails upon violation of
	//any of these
	if (bytes.Compare(file1.File, file2.File) != 0) ||
			(bytes.Compare(file2.File, file3.File) != 0) ||
			(bytes.Compare(file3.File, file4.File) != 0) ||
			(bytes.Compare(file4.File, file5.File) != 0) {
		return errors.New("inconsistency between log files in the first pass")
	}else{
		fmt.Println("no inconsistencies found in round 1")
		fmt.Println("passed test basic 5")
		return nil
	}
	fmt.Println("pass test basic 5")
	return nil
}


/**
 Simple majority check, if out of 5 servers 3 are dead the message should not get commtied!
 */
func testBasic6(cclient *chatclient.ChatClient, port1, port2, port3, port4, port5 int) error {
	argsKill1 := &multipaxos.SendMessageArgs{
		Value : make([]byte, 1),
		Tester : multipaxos.Tester{
			Stage : "sendPropose",
			Time : "start",
			Kill : true,
			SleepTime : 0,
		},
		PaxosPort : port1,
	}

	argsKill2 := &multipaxos.SendMessageArgs{
		Value : make([]byte, 1),
		Tester : multipaxos.Tester{
			Stage : "sendPropose",
			Time : "start",
			Kill : true,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}

	argsKill3 := &multipaxos.SendMessageArgs{
		Value : make([]byte, 1),
		Tester : multipaxos.Tester{
			Stage : "sendPropose",
			Time : "start",
			Kill : true,
			SleepTime : 0,
		},
		PaxosPort : port3,
	}

	err1 := cclient.SendMessage(argsKill1, &multipaxos.SendMessageReplyArgs{})
	if(err1 != nil){
		fmt.Println(err1)
		return err1
	}

	err2 := cclient.SendMessage(argsKill2, &multipaxos.SendMessageReplyArgs{})
	if(err2 != nil){
		fmt.Println(err2)
		return err2
	}

	err3 := cclient.SendMessage(argsKill3, &multipaxos.SendMessageReplyArgs{})
	if(err2 != nil){
		fmt.Println(err3)
		return err3
	}

	//Send message and make sure it is not commited because majority can't be reached
	msg := chatclient.ChatMessage{"alockwoo", "awesome", "poopmaster", time.Now(), ""}
	bytes, _ := json.Marshal(msg)

	args := &multipaxos.SendMessageArgs{
		Value : bytes,
		Tester : multipaxos.Tester{
			Stage : "" ,
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}

	cclient.SendMessage(args, &multipaxos.SendMessageReplyArgs{})

	time.Sleep(time.Second * 2)

	//Getting log files
	file1 := &multipaxos.FileReply{}
	file2 := &multipaxos.FileReply{}
	file3 := &multipaxos.FileReply{}
	file4 := &multipaxos.FileReply{}
	file5 := &multipaxos.FileReply{}

	errLogR1 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1)
	errLogR2 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2)
	errLogR3 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3)
	errLogR4 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4)
	errLogR5 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5)

	if (errLogR1 != nil || errLogR2 != nil || errLogR3 != nil ||
		errLogR4 != nil || errLogR5 != nil) {
		return errors.New("error occurred while getting the log file =| ")
	}

	//No messages should've been able to be committed
	if !(len(file1.File) == 0 && len(file2.File) == 0 && len(file3.File) == 0 &&
		len(file4.File) == 0 && len(file5.File) == 0) {

		return errors.New("no message shoulda been committed. also thanks demorgan")
	}

	return nil
}

/*
 * Check recovery of a node that dies before it sends any accept messages
 */
func testBasic7(cclient *chatclient.ChatClient, port1, port2, port3, port4, port5 int) error {

	msg1 := chatclient.ChatMessage{"alockwoo", "awesome", "poopmaster", time.Now(), ""}
	msg2 := chatclient.ChatMessage{"achaturv", "awesome", "golang ", time.Now(), ""}
	msg3 := chatclient.ChatMessage{"rebecca", "awesome", "~~im a troll~~!", time.Now(), ""}
	bytes1, err1 := json.Marshal(msg1)
	bytes2, err2 := json.Marshal(msg2)
	bytes3, err3 := json.Marshal(msg3)


	if(err1 != nil || err2 != nil || err3 != nil){
		return errors.New("error in testbasic7 while marshaling")
	}

	argsKill1 := &multipaxos.SendMessageArgs{
		Value : bytes1,
		Tester : multipaxos.Tester{
			Stage : "sendAccept",
			Time : "start",
			Kill : false,
			SleepTime : 4,
		},
		PaxosPort : port1,
	}

	cclient.SendMessage(argsKill1, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second)

	args2 := &multipaxos.SendMessageArgs{
		Value : bytes2,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}
	cclient.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})

	time.Sleep(time.Second * 4)

	args3 := &multipaxos.SendMessageArgs{
		Value : bytes3,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port4,
	}

	cclient.SendMessage(args3, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second * 2)

	//consistency of logs
	file1 := &multipaxos.FileReply{}
	file2 := &multipaxos.FileReply{}
	file3 := &multipaxos.FileReply{}
	file4 := &multipaxos.FileReply{}
	file5 := &multipaxos.FileReply{}

	errLogR1 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1)
	errLogR2 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2)
	errLogR3 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3)
	errLogR4 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4)
	errLogR5 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5)

	if (errLogR1 != nil || errLogR2 != nil || errLogR3 != nil ||
		errLogR4 != nil || errLogR5 != nil) {
		return errors.New("error occurred while getting the log file =| ")
	}

	fmt.Println("it got the files from all the servers! ")

	//check if message been committed to 1,2,3,4 and not to 5. fails upon violation of
	//any of these
	if (bytes.Compare(file1.File, file2.File) != 0) ||
			(bytes.Compare(file2.File, file3.File) != 0) ||
			(bytes.Compare(file3.File, file4.File) != 0) ||
			(bytes.Compare(file4.File, file5.File) != 0) {
		return errors.New("inconsistency between log files")
	}else{
		fmt.Println("no inconsistencies found in round 1")
		fmt.Println("passed test basic 5")
		return nil
	}
	return nil
}

/*
 * check recovery of a node that dies after sending accept messages
 */
func testBasic8(cclient *chatclient.ChatClient, port1, port2, port3, port4, port5 int) error {
	msg1 := chatclient.ChatMessage{"alockwoo", "awesome", "poopmaster", time.Now(), ""}
	msg2 := chatclient.ChatMessage{"achaturv", "awesome", "golang ", time.Now(), ""}
	msg3 := chatclient.ChatMessage{"skethu", "awesome", "i'm too cool", time.Now(), ""}

	bytes1, err1 := json.Marshal(msg1)
	bytes2, err2 := json.Marshal(msg2)
	bytes3, err3 := json.Marshal(msg3)

	if (err1 != nil || err2 != nil || err3 != nil) {
		return errors.New("error occurred while marshaling")
	}

	args1 := &multipaxos.SendMessageArgs{
		Value : bytes1,
		Tester : multipaxos.Tester{
			Stage : "sendAccept",
			Time : "end",
			Kill : false,
			SleepTime : 5,
		},
		PaxosPort : port1,
	}

	args2 := &multipaxos.SendMessageArgs{
		Value : bytes2,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}

	args3 := &multipaxos.SendMessageArgs{
		Value : bytes3,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port3,
	}

	cclient.SendMessage(args1, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second)

	cclient.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second*2)

	file1 := &multipaxos.FileReply{}
	file2 := &multipaxos.FileReply{}
	file3 := &multipaxos.FileReply{}
	file4 := &multipaxos.FileReply{}
	file5 := &multipaxos.FileReply{}

	errLogR1 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1)
	errLogR2 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2)
	errLogR3 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3)
	errLogR4 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4)
	errLogR5 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5)

	if (errLogR1 != nil || errLogR2 != nil || errLogR3 != nil ||
		errLogR4 != nil || errLogR5 != nil) {
		return errors.New("error occurred while getting the log file =| ")
	}

	fmt.Println("it got the files from all the servers! ")

	//check if message been committed to 1,2,3,4 and not to 5. fails upon violation of
	//any of these
	if (bytes.Compare(file1.File, file2.File) == 0) ||
			(bytes.Compare(file2.File, file3.File) != 0) ||
			(bytes.Compare(file3.File, file4.File) != 0) ||
			(bytes.Compare(file4.File, file5.File) != 0) {
		return errors.New("inconsistency between log files in the first pass")
	}

	time.Sleep(time.Second*3)

	cclient.SendMessage(args3, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second * 2)

	fileR1 := &multipaxos.FileReply{}
	fileR2 := &multipaxos.FileReply{}
	fileR3 := &multipaxos.FileReply{}
	fileR4 := &multipaxos.FileReply{}
	fileR5 := &multipaxos.FileReply{}

	errLogR1 = cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, fileR1)
	errLogR2 = cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, fileR2)
	errLogR3 = cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, fileR3)
	errLogR4 = cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, fileR4)
	errLogR5 = cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, fileR5)

	if (errLogR1 != nil || errLogR2 != nil || errLogR3 != nil ||
		errLogR4 != nil || errLogR5 != nil) {
		return errors.New("error occurred while getting the log file =| ")
	}

	fmt.Println("it got the files from all the servers! ")

	//check if message been committed to 1,2,3,4 and not to 5. fails upon violation of
	//any of these
	if (bytes.Compare(fileR1.File, fileR2.File) != 0) ||
			(bytes.Compare(fileR2.File, fileR3.File) != 0) ||
			(bytes.Compare(fileR3.File, fileR4.File) != 0) ||
			(bytes.Compare(fileR4.File, fileR5.File) != 0) {
		return errors.New("inconsistency between log files in the second pass")
	}
	return nil
}

/*
 * check recovery of a node that dies in the middle of sending commit messages
 */
func testBasic9(cclient *chatclient.ChatClient, port1, port2, port3, port4, port5 int) error {

	msg1 := chatclient.ChatMessage{"alockwoo", "awesome", "poopmaster", time.Now(), ""}
	msg2 := chatclient.ChatMessage{"achaturv", "awesome", "golang ", time.Now(), ""}

	bytes1, err1 := json.Marshal(msg1)
	bytes2, err2 := json.Marshal(msg2)

	if (err1 != nil || err2 != nil) {
		return errors.New("error occurred while marshaling")
	}

	args1 := &multipaxos.SendMessageArgs{
		Value : bytes1,
		Tester : multipaxos.Tester{
			Stage : "sendCommit",
			Time : "mid",
			Kill : false,
			SleepTime : 3,
		},
		PaxosPort : port1,
	}

	args2 := &multipaxos.SendMessageArgs{
		Value : bytes2,
		Tester : multipaxos.Tester{
			Stage : "",
			Time : "",
			Kill : false,
			SleepTime : 0,
		},
		PaxosPort : port2,
	}

	cclient.SendMessage(args1, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second * 4)

	cclient.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})
	time.Sleep(time.Second)

	file1 := &multipaxos.FileReply{}
	file2 := &multipaxos.FileReply{}
	file3 := &multipaxos.FileReply{}
	file4 := &multipaxos.FileReply{}
	file5 := &multipaxos.FileReply{}

	errLogR1 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port1, }, file1)
	errLogR2 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port2, }, file2)
	errLogR3 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port3, }, file3)
	errLogR4 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port4, }, file4)
	errLogR5 := cclient.GetLogFile(&multipaxos.FileArgs{Port : port5, }, file5)

	if (errLogR1 != nil || errLogR2 != nil || errLogR3 != nil ||
		errLogR4 != nil || errLogR5 != nil) {
		return errors.New("error occurred while getting the log file =| ")
	}

	fmt.Println("it got the files from all the servers! ")

	//check if message been committed to 1,2,3,4 and not to 5. fails upon violation of
	//any of these
	if (bytes.Compare(file1.File, file2.File) != 0) ||
			(bytes.Compare(file2.File, file3.File) != 0) ||
			(bytes.Compare(file3.File, file4.File) != 0) ||
			(bytes.Compare(file4.File, file5.File) != 0) {
		return errors.New("inconsistency between log files in the first pass")
	}

	return nil
}

func main(){
	isMaster := flag.Bool("isMaster", false, "to check if its a master server or not ")

	N := flag.Int("N", 0, "to specify the number of servers")
	port := flag.Int("port", 1111, "to specify the port number to start the master server on")
	testNum := flag.Int("testNum", 0, "the number of the test we want to run")
	registerAll := flag.Bool("registerAll", false, "start test cases once all servers have been registered")
	flag.Parse()


	//CALL TESTS
	if *registerAll {
		time.Sleep(time.Second*5)
		var err error

		fmt.Println("STARTTING A NEW CHAT CLIENT")
		cClient, _ := chatclient.NewChatClient("2000", 8080)

		if *testNum == 0 {
			fmt.Println("TEST GET_SERVERS")
			err = TestGetServers(cClient)
		}

		if *testNum == 1 {
			fmt.Println("TEST_BASIC_1")
			err = testBasic1(cClient)
		}

		if *testNum == 2 {
			fmt.Println("TEST_BASIC_2")
			err = testBasic2(cClient, 8080, 9990)
		}

		if *testNum == 3 {
			fmt.Println("TEST_BASIC_3")
			err = testBasic3(cClient, 8080, 9990, 8081)
		}

		if *testNum == 4 {
			fmt.Println("TEST_BASIC_4")
			err = testBasic4(cClient, 8080, 8081, 8082, 8083, 9990)
		}

		if *testNum == 5 {
			fmt.Println("TEST_BASIC_5")
			err = testBasic5(cClient, 8080, 9990, 8081, 8082, 8083)
		}

		if *testNum == 6 {
			fmt.Println("TEST_BASIC_6")
			err = testBasic6(cClient, 8080, 8081, 8082, 8083, 9990)
		}

		if *testNum == 7 {
			fmt.Println("TEST_BASIC_7")
			err = testBasic7(cClient, 8080, 8081, 8082, 8083, 9990)
		}

		if *testNum == 8 {
			fmt.Println("TEST_BASIC_8")
			err = testBasic8(cClient, 8080, 8081, 8082, 8083, 9990)
		}

		if *testNum == 9 {
			fmt.Println("TEST_BASIC_9")
			err = testBasic9(cClient, 8080, 8081, 8082, 8083, 9990)
		}

		if err != nil {
			fmt.Println("FAIL", err)
		} else {
			fmt.Println("PASS")
		}

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
