package chatclient

import (
	"code.google.com/p/go.net/websocket"
	"fmt"  
	"container/list"
	"github.com/cmu440/chatterbox/multipaxos"
	"errors"
	"encoding/json"
)

type User struct{
	Connection *websocket.Conn
	Name string 
	Send chan Message
	Rooms *list.List //list of rooms a user is joined to
	chanMessage chan string
}

type ChatRoom struct{
	Users map[string]*User
	Broadcast chan Message

}

type Message struct{
	User *User
	Time string
	Contents string
}

var Rooms *list.List		//list of Chatroom objects
var Users map[string] User 	//list of UserObjects
var ClientConn *rpc.Client
var PaxosServers []multipaxos.PaxosServer
var Hostport string

func NewChatClient(hostport string) {
	//TODO setup ClientConn, and Paxos Servers

}

//TODO called by the http.Handler when we set up the rendering stuff
func NewUser(ws *websocket.Conn) error {

	username := ws.Request().URL.Query().Get("username")

	if username == "" {
		err := errors.New("invalid input for user")
		marshalled, err := json.Marshal(err)
		ws.Write(marshalled)
		return err
	}

	joiningUser := &User{
		Connection : ws,
		Name : username,
	}

	Users[username] = joiningUser

	go joiningUser.GetInfoFromUser(ws)
	go joiningUser.GetSendMessagesToUser()

	return nil
}

func (user *User) GetInfoFromUser (ws *websocket.Conn) {
	for {
		//TODO RECEIVE messages from user and if the message is to join a new room update stats
		//else if its  a message call SendMessage
	}
}

func (user *User) SendMessagesToUser () {
	for {
		//TODO every 2 seconds get the logs and get diff and send new messages to the gui 
	}
}


func SendMessage(args *multipaxos.SendMessageArgs, reply *multipaxos.SendMessageReplyArgs) error {
	fmt.Println("Sending Message in Chat Client")
	errCall := ClientConn.Call("PaxosServer.SendMessage", &args, &reply)
	return errCall
}

func  GetServers(args *multipaxos.GetServersArgs, reply*multipaxos.GetServersReply) error {
	return ClientConn.Call("PaxosServer.GetServers", &args, &reply)
}
