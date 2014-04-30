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
	"io/ioutil"
	"html/template"
	"os"
	"strings"
	"bytes"
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
var ChanMessages chan *multipaxos.SendMessageArgs
var ChanCommitedMessages chan *multipaxos.SendMessageArgs

func NewChatClient(port string, paxosPort int) (*ChatClient, error){
	fmt.Println("Starting a New Chat Client")
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
	ChanMessages = make(chan *multipaxos.SendMessageArgs)


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
	fmt.Println("Set the Paxos Servers", len(PaxosServers))

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

	//http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/css/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL.Path[5:])
			f, err := os.Open(r.URL.Path[5:])

			if err != nil {
				fmt.Println("blabjalh", err)
			}

			http.ServeContent(w, r, ".css", time.Now(), f)
		})
	http.HandleFunc("/js/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL.Path[4:])
			f, err := os.Open(r.URL.Path[4:])

			if err != nil {
				fmt.Println("alkdjfalkfd", err)
			}

			http.ServeContent(w, r, ".js", time.Now(), f)
		})
	http.HandleFunc("/images/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL.Path[8:])
			f, err := os.Open(r.URL.Path[8:])

			if err != nil {
				fmt.Println("alkdjfalkfd", err)
			}

			http.ServeContent(w, r, ".jpg", time.Now(), f)
		})

	http.HandleFunc("/", startPageHandler)
	http.Handle("/join", websocket.Handler(NewUser))

	fmt.Println("Finished Creating New Chat Client")
	return chatclient, nil
}

func startPageHandler(w http.ResponseWriter, r *http.Request){
	fmt.Println("in start page handler")
	t, _ := template.ParseFiles("startPage.html")
	if t == nil {
		fmt.Println("why is t nil?")
	}
	if w == nil {
		fmt.Println("why is w nil?")
	}
	t.Execute(w, nil)
}

func (cc *ChatClient) FailedMessages() {
	for {
		select {
		case req := <-ChanMessages:
			fmt.Println("Trying again to send the message")
			cc.SendMessage(req, &multipaxos.SendMessageReplyArgs{})
		}
	}

}

//TODO called by the http.Handler when we set up the rendering stuff
func NewUser(ws *websocket.Conn) {
	fmt.Println("Creating New User")
	username := ws.Request().URL.Query().Get("username")
	fmt.Println("Username is", username)

	if username == "" {
		err := errors.New("invalid input for user")
		marshalled, err := json.Marshal(err)
		ws.Write(marshalled)
		fmt.Println(err)
		return
	}

	fmt.Println("HERE")
	joiningUser := &User{
		Connection : ws,
		Name : username,
		Rooms : list.New(),
		TimeRecd : time.Now(),
	}

	if _, ok := Users[username]; ok {
		//TODO handle the case when the user is already in chatterbox
	}

	Users[username] = joiningUser
	cc := &ChatClient{}

	go joiningUser.GetInfoFromUser(ws)
	go cc.FailedMessages()
	joiningUser.SendMessagesToUser()

}

func (user *User) GetInfoFromUser (ws *websocket.Conn) {
	fmt.Println("Inside GETINFOFROMUSER")
	for {
		//TODO RECEIVE messages from user and if the message is to join a new room update stats
		//else if its  a message call SendMessage
		var content string
		err := websocket.Message.Receive(user.Connection, &content)
		fmt.Println("Received a message", content)

		if err != nil {
			fmt.Println("err while receiving a message!", err)
			continue
		}

		msg := ChatMessage {
			User: user.Name,
			Room:  "still working on this",	//TODO
			Content: content,
			Timestamp:  time.Now(),
		}
		msgString, err := msg.ToString()
		if err!= nil {
			fmt.Println("Why Couldn't message become a string", msgString , err)
		}
		fmt.Println(msgString)

		randPort := PaxosServers[rand.Int()%len(PaxosServers)]
		fmt.Println("Sending a message to", randPort)

		bytes, marshalErr := json.Marshal(msg)
		if marshalErr != nil {
			fmt.Println("Couldn't marshall the message", marshalErr)
			continue
		}

		args := &multipaxos.SendMessageArgs{Value : bytes,
			Tester : multipaxos.Tester{
				Stage : "",
				Time : "",
				Kill : false,
				SleepTime : 0,
			},
			PaxosPort: randPort}
		reply := &multipaxos.SendMessageReplyArgs{}

		cc := &ChatClient{}
		cc.SendMessage(args, reply)

	}
}

func (user *User) SendMessagesToUser() error{
	fmt.Println("Inside SENDMESSAGESTOUSER")
	for {

		time.Sleep(time.Second*5)
		fmt.Println("Trying to get log files")
		randPort := PaxosServers[rand.Int()%len(PaxosServers)]
		conn := PaxosServerConnections[randPort]
		args := &multipaxos.CommitReplyArgs{}
		reply := &multipaxos.FileReply{}

		errCall := conn.Call("PaxosServer.ServeMessageFile", &args, &reply)
		fmt.Println("Finished call")

		if(errCall != nil){
			fmt.Println(errCall)
			return errCall
		}

		var err error
		fmt.Println("Reading the lines right now")

		msgs := strings.Split(string(reply.File), "}{")
		for i := 0; i < len(msgs); i++{
			//loss of characters during string split
			if len(msgs) > 1 {
				if(i == 0){
					msgs[i] = msgs[i]+ "}"
				}else if i == len(msgs) -1 {
					msgs[i] = "{" + msgs[i]
				} else {
					msgs[i] = "{" + msgs[i]+ "}"
				}
			}
			message := &ChatMessage{}
			byteMsgs := []byte(msgs[i])
			err = json.Unmarshal(byteMsgs, message)
			messageTime := message.Timestamp
			message.TimeString = message.Timestamp.Format(time.Kitchen)
			msgBytes, _ := json.Marshal(message)
			if err != nil {
				fmt.Println("Couldn't unmarshsal to create the message")
			}
			fmt.Println("User time is", user.TimeRecd)
			fmt.Println("Message time is", message.Timestamp)
			if(messageTime.After(user.TimeRecd)){
				fmt.Println("Sending to js now")
				user.Connection.Write(bytes.Trim(msgBytes, "\x00"))
				user.TimeRecd = messageTime
			}
		}
		fmt.Println(err)
	}
	return nil
}

type Page struct{
	Title string
	Body []byte
}

func loadPage(title string) (*Page, error) {
	body, err := ioutil.ReadFile(title)
	if(err != nil){
		fmt.Println("Couldn't get file", err)
		return nil, err
	}
	return &Page{
		Title: title,
		Body: body,
	}, nil
}


func (cc *ChatClient)SendMessage(args *multipaxos.SendMessageArgs, reply *multipaxos.SendMessageReplyArgs) error {
	go func() {
		fmt.Println("Sending Message in Chat Client", args.PaxosPort)
		conn := PaxosServerConnections[args.PaxosPort]

		errCall := conn.Call("PaxosServer.SendMessage", &args, &reply)
		if errCall != nil {
			fmt.Println(args.PaxosPort, errCall)
			ChanMessages <- args
		} else {
			ChanCommitedMessages <- args
			fmt.Println("Successfully commited message", args.PaxosPort)
		}

	}()

	return nil
}

func (cc *ChatClient)GetServers(args *multipaxos.GetServersArgs, reply*multipaxos.GetServersReply) error {
	return ClientConn.Call("PaxosServer.GetServers", args, reply)
}


func (cc *ChatClient) GetLogFile(args *multipaxos.FileArgs, reply *multipaxos.FileReply) error{
	fmt.Println("in getlogfile for port number ", args.Port)
	conn := PaxosServerConnections[args.Port]
	err := conn.Call("PaxosServer.ServeMessageFile", args, reply)
	if(err != nil){
		fmt.Println(err)
		return err
	}
	return nil
}

func (cc *ChatClient) GetRooms() error {
	//TODO for testing
	return errors.New("Unimplemented")
}

func (cc *ChatClient) GetUsers() error {
	//TODO for testing
	return errors.New("Unimplemented")
}
