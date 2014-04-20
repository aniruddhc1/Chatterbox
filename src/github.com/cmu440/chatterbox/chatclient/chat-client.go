package chatclient

import (
	"errors"
	"net/rpc"
	"fmt"
	"net"
	"net/http"
)



type ChatClient struct {
	ClientConn *rpc.Client
}

type InputArgs struct {
	Value string
}

type OutputArgs struct {

}

func NewChatClient(hostport string) (*ChatClient, error) {
	//TODO unimplemented

	chatclient := ChatClient{}

	errRegister := rpc.RegisterName("ChatClient", &chatclient)
	if errRegister != nil {
		fmt.Println("Couldln't register test chat client", errRegister)
		return nil, errRegister
	}

	rpc.HandleHTTP()
	listener, errListen := net.Listen("tcp", hostport)

	if errListen != nil {
		fmt.Println("Couldln't listen test chat client", errListen)
		return nil, errListen
	}

	go http.Serve(listener, nil)

	chatConn, errDial := rpc.DialHTTP("tcp", "localhost:8080")
	if errDial != nil {
		fmt.Println("Couldln't dialtest chat client", errDial)
		return nil, errDial
	}

	chatclient.ClientConn = chatConn

	return &chatclient, nil
}

func (*ChatClient) CreateNewUser(args *InputArgs, reply *OutputArgs) error {
	return errors.New("Not Implemented")
}

func (*ChatClient) JoinChatRoom(args *InputArgs, reply *OutputArgs) error {
	return errors.New("Not Implemented")
}

func (*ChatClient) SendMessage(args *InputArgs, reply *OutputArgs) error {
	return errors.New("Not Implemented")
}
