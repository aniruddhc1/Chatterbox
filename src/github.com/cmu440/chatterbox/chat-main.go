package main

import (
	"github.com/cmu440/chatterbox/chatclient"
	"fmt"
	"flag"
	"github.com/cmu440/chatterbox/multipaxos"
	"strconv"
	"time"
)


func main() {

	isMaster := flag.Bool("isMaster", false, "to check if its a master server or not")
	startChat := flag.Bool("startChat", false, "start test cases once all servers have been registered")

	N := flag.Int("N", 0, "to specify the number of servers")
	port := flag.Int("port", 1111, "to specify the port number to start the master server on")

	flag.Parse()

	if *startChat {
		time.Sleep(time.Second *3)
		fmt.Println(*port)
		_, err := chatclient.NewChatClient(strconv.Itoa(*port), 8080)

		if err != nil {
			fmt.Println("Couldnt start a new chat client - Fail")
			return
		}

		select{}
	} else if *isMaster {
		_, err := multipaxos.NewPaxosServer("", *N, *port) //starting master server
		if(err != nil){
			fmt.Println(err)
		}
		select{}
	} else {
		_, err := multipaxos.NewPaxosServer("localhost:8080", *N, *port) // starting node

		if(err != nil){
			fmt.Println(err)
		}
		select{}
	}

}
