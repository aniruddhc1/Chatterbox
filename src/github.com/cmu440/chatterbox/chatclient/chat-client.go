package chatclient

import (
	"errors"
	"net/rpc"
	"fmt"
	"net"
	"net/http"
	"container/list"
	"github.com/cmu440/chatterbox/multipaxos"
)

type User struct{ //wrapper struct for extensibility
	Username string
	Rooms *list.List
}

type Room struct{ //wrapper struct for extensibility
	Name string
	Users *list.List
}

type ChatClient struct {
	ClientConn *rpc.Client
	Users *list.List //list of all users
	Rooms *list.List //list of all chat rooms

}

type InputArgs struct {
	Value string
}

type OutputArgs struct {

}

func NewChatClient(hostport string) (*ChatClient, error) {
	//TODO unimplemented

	chatclient := &ChatClient{
		Users : list.New(),
		Rooms : list.New(),
	}

	errRegister := rpc.RegisterName("ChatClient", Wrap(chatclient))
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
		fmt.Println("Couldn't dialtest chat client", errDial)
		return nil, errDial
	}

	chatclient.ClientConn = chatConn

	return chatclient, nil
}

func (cc *ChatClient) CreateNewUser(args *InputArgs, reply *OutputArgs) error { //needs user and room



	return errors.New("Not Implemented")



}

func (cc *ChatClient) JoinChatRoom(args *InputArgs, reply *OutputArgs) error { //needs user and room
	return errors.New("Not Implemented")
}

func (cc *ChatClient) SendMessage(args *multipaxos.SendMessageArgs, reply *multipaxos.SendMessageReplyArgs) error {
	errCall := cc.ClientConn.Call("PaxosServer.GetServers", &args, &reply)
	return errCall
}

func (cc *ChatClient) GetServers(args *multipaxos.GetServersArgs, reply*multipaxos.GetServersReply) error {
	return cc.ClientConn.Call("PaxosServer.GetServers", &args, &reply)
}
