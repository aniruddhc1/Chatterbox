package chatclient

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"container/list"
	"github.com/cmu440/chatterbox/multipaxos"
	"errors"
	"net/rpc"
	"net/http"
	"strconv"
	"net"
	"encoding/json"
	"time"
	"math/rand"
	"bufio"
)

type User struct{
	Connection *websocket.Conn
	Name string
	Rooms *list.List //list of rooms a user is joined to
	TimeRecd time.Time
}

type ChatRoom struct{
	Users map[string]*User
}

type Message struct{
	room ChatRoom
	User *User
	Time string
	Contents string
}

type ChatClient struct {

}


var Rooms *list.List		//list of Chatroom objects
var Users map[string] *User 	//list of UserObjects
var ClientConn *rpc.Client
var PaxosServers []int
var Hostport string
var PaxosServerConnections map[int] *rpc.Client

func NewChatClient(port string, paxosPort int) (*ChatClient, error){

	//TODO setup ClientConn, and Paxos Servers
	chatclient := &ChatClient{}

	errRegister := rpc.RegisterName("ChatClient", Wrap(chatclient))
	if errRegister != nil {
		fmt.Println("Couldln't register test chat client", errRegister)
		return nil, errRegister
	}

	rpc.HandleHTTP()
	listener, errListen := net.Listen("tcp", ":"+port)

	if errListen != nil {
		fmt.Println("Couldln't listen test chat client", errListen)
		return nil, errListen
	}

	go http.Serve(listener, nil)

	chatConn, errDial := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(paxosPort))
	if errDial != nil {
		fmt.Println("Couldn't dialtest chat client", errDial)
		return nil, errDial
	}

	//INITIALIZING GLOBAL VARIABLES
	Rooms = list.New()		//list of Chatroom objects
	Users = make(map[string] *User) 	//list of UserObjects
	PaxosServerConnections = make(map[int] *rpc.Client)
	ClientConn = chatConn


	//GETTING ALL PAXOS SERVER CONNECTIONS
	getServerArgs :=  &multipaxos.GetServersArgs{}
	getServerReply := &multipaxos.GetServersReply{}

	err := chatclient.GetServers(getServerArgs, getServerReply)
	if err != nil {
		fmt.Println("Error occured while getting servers", err)
		return nil, err
	}

	//TODO do the weird waiting thing, for now assume that it is done
	PaxosServers = getServerReply.Servers

	for i := 0; i < len(PaxosServers); i++ {
		fmt.Println("Adding rpc connections for", PaxosServers[i])
		currPort := PaxosServers[i]

		serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(currPort))
		if dialErr != nil {
			fmt.Println("Error occured while dialing to all servers", dialErr)
			return nil, dialErr
		} else {
			PaxosServerConnections[currPort] = serverConn
		}
	}

	return chatclient, nil
}

func (cc *ChatClient) GetRooms() error {
	//TODO for testing
	return errors.New("Unimplemented")
}

func (cc *ChatClient) GetUsers() error {
	//TODO for testing
	return errors.New("Unimplemented")
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
		Rooms : list.New(),
		TimeRecd : time.Now(),
	}

	Users[username] = joiningUser

	//go joiningUser.GetInfoFromUser(ws)
	//go joiningUser.SendMessagesToUser()

	return nil
}

func (user *User) GetInfoFromUser (ws *websocket.Conn) {
	for {
		//TODO RECEIVE messages from user and if the message is to join a new room update stats
		//else if its  a message call SendMessage


	}
}

func (user *User) SendMessagesToUser() error{
	for {
		//TODO every 2 seconds get the logs and get diff and send new messages to the gui
		time.Sleep(time.Second*2)
		randPort := PaxosServers[rand.Int()%len(PaxosServers)]
		conn := PaxosServerConnections[randPort]
		args := &multipaxos.CommitReplyArgs{}
		reply := &multipaxos.FileReply{}

		errCall := conn.Call("PaxosServer.ServeMessageFile", &args, &reply)
		if(errCall != nil){
			fmt.Println(errCall)
			return errCall
		}
		msgFile := reply.File
		reader := bufio.NewReader(msgFile)

		var err error
		var line []byte
		for err == nil {
			line, err = reader.ReadBytes('\n')
			msg := &ChatMessage{}
			json.Unmarshal(line, msg)
			if(msg.Timestamp.After(user.TimeRecd)){
				//TODO send message to chat client over websocket
			}
		}
		fmt.Println(err)
		user.TimeRecd = time.Now()
	}
	return nil
}


func (cc *ChatClient)SendMessage(args *multipaxos.SendMessageArgs, reply *multipaxos.SendMessageReplyArgs) error {
	fmt.Println("Sending Message in Chat Client")
	conn := PaxosServerConnections[args.PaxosPort]
	errCall := conn.Call("PaxosServer.SendMessage", &args, &reply)
	return errCall
}

func (cc *ChatClient)GetServers(args *multipaxos.GetServersArgs, reply*multipaxos.GetServersReply) error {
	return ClientConn.Call("PaxosServer.GetServers", &args, &reply)
}
