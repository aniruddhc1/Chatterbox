package main

import (
	"fmt"
	"errors"
	"flag"
	"time"
	"github.com/cmu440/chatterbox/chatclient"
	"encoding/json"
	"github.com/cmu440/chatterbox/multipaxos"
	"bufio"
	"io/ioutil"
	"bytes"
	"os"
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
	msg := chatclient.ChatMessage{"Soumya", "testRoom", "TESTING", time.Now()}

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
	return nil
}



/*
 * Does one iteration of paxos with the failure of first proposer after sending
 * accept messages. Test that it commits the right message (the first one should be
 * commited)
 */
func testBasic2(cClient1 *chatclient.ChatClient, port1 int, port2 int) error {
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

	//look at file before

	fileArgs := &multipaxos.FileArgs{
		Port : port1,
	}
	fileReply1 := &multipaxos.FileReply{}

	errLog1 := cClient1.GetLogFile(fileArgs, fileReply1)
	if(errLog1 != nil){
		return errLog1
	}

	err := cClient1.SendMessage(args1, &multipaxos.SendMessageReplyArgs{})

	fileReply2 := &multipaxos.FileReply{}
	cClient1.GetLogFile(fileArgs, fileReply2)

	var fileU1 *os.File
	var fileU2 *os.File
	errU1 := json.Unmarshal(fileReply1.File, fileU1)
	errU2 := json.Unmarshal(fileReply2.File, fileU2)
	if(errU1 != nil || errU2 != nil ||
			(len(fileReply1.File) == 0) || (len(fileReply2.File) == 0) ){
		fmt.Println("error in unmarshal")
		return errors.New("error in unmarshal")
	}

	replyBytes1, err := ioutil.ReadAll(bufio.NewReader(fileU1))
	if(err != nil){
		return err
	}

	replyBytes2, err2 := ioutil.ReadAll(bufio.NewReader(fileU2))
	if(err2 != nil){
		return err2
	}

	if(bytes.Compare(replyBytes1, replyBytes2) != 0){
		return errors.New("commit went through in spite of killing the server after sendAccept phase")
	}

	//look at file after and do a diff

	if(err != nil){
		return err
	}

	error2 := cClient1.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})
	if(error2 != nil){
		return error2
	}

	fileReply3 := &multipaxos.FileReply{}
	cClient1.GetLogFile(fileArgs, fileReply3)

	var fileU3 *os.File
	errU3 := json.Unmarshal(fileReply1.File, fileU1)
	if(errU3 != nil || len(fileReply1.File) == 0){
		fmt.Println("error in unmarshal")
		return errors.New("error in unmarshal")
	}

	replyBytes3, err3 := ioutil.ReadAll(bufio.NewReader(fileU3))

	if(err3 != nil){
		return err3
	}

	if(bytes.Compare(replyBytes2, replyBytes3) == 0){
		return errors.New("nothing added to the log file!!")
	} else{
		var line3 string
		var err3 error
		for {
			var fileU4 *os.File
			errU1 := json.Unmarshal(fileReply1.File, fileU4)
			if(errU1 != nil || len(fileReply1.File) == 0){
				fmt.Println("error in unmarshal")
				return errors.New("error in unmarshal")
			}
			line3, err3 = bufio.NewReader(fileU4).ReadString('\n')
			if err3 != nil {
				break
			}
		}
		Line3, _ := json.Marshal(line3)
		if bytes.Compare(Line3, replyBytes2) == 0 && bytes.Compare(Line3, replyBytes1) != 0 {
			errors.New("message 1 did not get added to the file!")
		}
	}

	//TODO this is working but Ani will create a function that will look at the log file before first send message
	//make sure that after this send message file is not modified
	//then make sure that after second send message that msg1 is in file not msg2!

	return nil
}


/*
 *
 */
func testBasic3(cClient1 *chatclient.ChatClient, port1 int, port2 int, port3 int) error {
	msg1 := chatclient.ChatMessage{"Srini", "testRoom", "lololollol :D", time.Now()}
	bytes1, marshalErr := json.Marshal(msg1)
	if(marshalErr != nil){
		return errors.New("error occurred while marshaling msg1 in testBasic2")
	}

	msg2 := chatclient.ChatMessage{"Taran", "testRoom", "get out hahaa", time.Now()}
	bytes2, marshalErr1 := json.Marshal(msg2)
	if(marshalErr1 != nil){
		return errors.New("error occurred while marshaling msg2 in testBasic2")
	}


	msg3 := chatclient.ChatMessage{"Sally", "testRoom", "cool stories", time.Now()}
	bytes3, marshalErr2 := json.Marshal(msg3)
	if(marshalErr2 != nil){
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
	if(err2 != nil){
		fmt.Println("Error 2", err2)
		return err2
	}

	time.Sleep(time.Second*20)

	err3 := cClient1.SendMessage(args3, &multipaxos.SendMessageReplyArgs{})
	if(err3 != nil){
		fmt.Println("Error 3", err3)
		return err3
	}

	time.Sleep(time.Second*2)
	//TODO send the first one make and it wait right before it sends accept requests
	//then do send args 2 while its waiting
	//then wait for like 3 seconds nd try
	//sending args 3

	return nil
}

func testBasic4(cclient *chatclient.ChatClient, port1, port2, port3, port4, port5 int) error {
	msg1 := chatclient.ChatMessage{"alockwoo", "awesome", "poopmaster", time.Now()}
	msg2 := chatclient.ChatMessage{"achaturv", "awesome", "golang ", time.Now()}
	msg3 := chatclient.ChatMessage{"skethu", "awesome", "i'm too cool", time.Now()}
	msg4 := chatclient.ChatMessage{"cgarrod", "awesome", "i love webapps!", time.Now()}


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

	er := cclient.SendMessage(argsKill, &multipaxos.SendMessageReplyArgs{})
	if(er != nil){
		fmt.Println(er)
		return er
	}

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

	er1 := cclient.SendMessage(argsDelay, &multipaxos.SendMessageReplyArgs{})

	if er1 != nil{
		fmt.Println(er1)
		return er1
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

	errSend2 := cclient.SendMessage(args2, &multipaxos.SendMessageReplyArgs{})

	if(errSend2 != nil){
		fmt.Println(errSend2)
		return errSend2
	}

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

	if (bytes.Compare(file1R2.File, file2R2.File) != 0) ||
			(bytes.Compare(file2R2.File, file3R2.File) != 0) ||
			(bytes.Compare(file3R2.File, file4R2.File) == 0) ||
			(bytes.Compare(file4R2.File, file5R2.File) == 0) {
		return errors.New("inconsistency between log files in the second round")
	}

	fmt.Println("passed round 2")

	//wait till delay ends.. then make 4 send a proposal, send message.
	//make sure that 1,2,3,4 is consistent

	time.Sleep(time.Second * 2)

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

func testBasic5(cclient *chatclient.ChatClient, port1, port2, port3, port4, port5 int) error{
	msg1 := chatclient.ChatMessage{"alockwoo", "awesome", "poopmaster", time.Now()}
	msg2 := chatclient.ChatMessage{"achaturv", "awesome", "golang ", time.Now()}
	msg3 := chatclient.ChatMessage{"skethu", "awesome", "i'm too cool", time.Now()}
	msg4 := chatclient.ChatMessage{"cgarrod", "awesome", "i love webapps!", time.Now()}
	msg5 := chatclient.ChatMessage{"rebecca", "awesome", "~~im a troll~~!", time.Now()}

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

func main(){
	isMaster := flag.Bool("isMaster", false, "to check if its a master server or not ")
	registerAll := flag.Bool("registerAll", false, "start test cases once all servers have been registered")

	N := flag.Int("N", 0, "to specify the number of servers")
	port := flag.Int("port", 1111, "to specify the port number to start the master server on")
	flag.Parse()


	//CALL TESTS
	if *registerAll {
		time.Sleep(time.Second*15)

		fmt.Println("STARTTING A NEW CHAT CLIENT")
		cClient, _ := chatclient.NewChatClient("2000", 8080)

//		fmt.Println("TEST GET_SERVERS")
//		err := TestGetServers(cClient)
//		fmt.Println(err)

//		fmt.Println("TEST_BASIC_1")
//		err1 := testBasic1(cClient)
//		fmt.Println(err1)

//		fmt.Println("TEST_BASIC_2")
//		err2 := testBasic2(cClient, 8080, 9990)
//		fmt.Println(err2)


//		fmt.Println("TEST_BASIC_3")
//		err3 := testBasic3(cClient, 8080, 9990, 8081)
//		fmt.Println(err3)

//		fmt.Println("TEST_BASIC_4")
//		err4 := testBasic4(cClient, 8080, 9990, 8081, 8082, 8083)
//		fmt.Println(err4)

		fmt.Println("TEST_BASIC_5")
		err4 := testBasic5(cClient, 8080, 9990, 8081, 8082, 8083)
		fmt.Println(err4)


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
