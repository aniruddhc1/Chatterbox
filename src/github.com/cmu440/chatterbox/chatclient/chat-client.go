package chatclient
//
//import (
//	"errors"
//	"net/rpc"
//	"fmt"
//	"net"
//	"net/http"
//	"container/list"
//	"github.com/cmu440/chatterbox/multipaxos"
//	"strconv"
//	"code.google.com/p/go.net/websocket"
//	"os"
//	"bufio"
//	"time"
//	"github.com/cmu440/chatterbox"
//)
//
//type User struct{ //wrapper struct for extensibility
//	Username string
//	Rooms *list.List
//	ActiveRoom Room
//	Connection *websocket.Conn
//}
//
//type Room struct{ //wrapper struct for extensibility
//	Name string
//	Users *list.List
//	//option to close the chat room?
//}
//
//type ChatClient struct {
//	ClientConn *rpc.Client
//	Users *list.List //list of all users
//	Rooms *list.List //list of all chat rooms
//
//}
//
//type InputArgs struct {
//	ws *websocket.Conn
//}
//
//type OutputArgs struct {
//
//}
//
//var currentRoom *Room = &Room{}
//
//func NewChatClient(hostport string, paxosPort int) (*ChatClient, error) {
//	//TODO unimplemented
//
//	chatclient := &ChatClient{
//		Users : list.New(),
//		Rooms : list.New(),
//	}
//
//	errRegister := rpc.RegisterName("ChatClient", Wrap(chatclient))
//	if errRegister != nil {
//		fmt.Println("Couldln' t register test chat client", errRegister)
//		return nil, errRegister
//	}
//
//	rpc.HandleHTTP()
//	listener, errListen := net.Listen("tcp", hostport)
//
//	if errListen != nil {
//		fmt.Println("Couldln't listen test chat client", errListen)
//		return nil, errListen
//	}
//
//	go http.Serve(listener, nil)
//
//	chatConn, errDial := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(paxosPort))
//	if errDial != nil {
//		fmt.Println("Couldn't dialtest chat client", errDial)
//		return nil, errDial
//	}
//
//	chatclient.ClientConn = chatConn
//
//	return chatclient, nil
//}
//
//func (cc *ChatClient) CreateNewUser(args *InputArgs, reply *OutputArgs) error { //needs user and room
//	username := args.ws.Request().URL.Query().Get("username")
//
//	if username == ""{
//		return errors.New("username is invalid")
//	}
//
//	user := &User{
//		Username : username,
//		Connection : args.ws,
//		ActiveRoom : currentRoom,
//		Rooms : list.New().PushBack(currentRoom),
//	}
//
//	currentRoom.Users.PushBack(user)
//
//	cc.UpdateView(&OutputArgs{}, &UpdateViewReply{})
//
//
//
//}
//
//
//func NewChatRoom(){
//	currentRoom := &Room{
//		Name : "",
//		Users : list.New(),
//	}
//	return currentRoom
//}
//
//type UpdateViewReply struct{
//	file os.File
//}
//
//func (cc *ChatClient) UpdateView(args *OutputArgs, reply *UpdateViewReply){
//	inputFile := bufio.NewScanner(reply.file)
//	for {
//		line := inputFile.ReadLine()
//		fmt.Println(line)
//		time.Sleep(time.Second)
//	}
//}
//
//func (cc *ChatClient) JoinChatRoom(args *InputArgs, reply *OutputArgs) error { //needs user and room
//	return errors.New("Not Implemented")
//}
//
//func (cc *ChatClient) SendMessage(args *multipaxos.SendMessageArgs, reply *multipaxos.SendMessageReplyArgs) error {
//	fmt.Println("Sending Message in Chat Client")
//	errCall := cc.ClientConn.Call("PaxosServer.SendMessage", &args, &reply)
//	return errCall
//}
//
//func (cc *ChatClient) GetServers(args *multipaxos.GetServersArgs, reply*multipaxos.GetServersReply) error {
//	return cc.ClientConn.Call("PaxosServer.GetServers", &args, &reply)
//}

