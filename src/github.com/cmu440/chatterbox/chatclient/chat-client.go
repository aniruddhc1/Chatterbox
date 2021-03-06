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
	FirstTime bool
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
	//Starting a New Chat Client
	chatclient := &ChatClient{}

	errRegister := rpc.RegisterName("ChatClient", Wrap(chatclient))
	if errRegister != nil {
		fmt.Println("Couldln't register test chat client", errRegister)
		return nil, errRegister
	}

	rpc.HandleHTTP()
	listener, errListen := net.Listen("tcp", ":"+port)

	if errListen != nil {
		fmt.Println("Couldn't listen to test chat client", errListen)
		return nil, errListen
	}

	go http.Serve(listener, nil)

	chatConn, errDial := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(paxosPort))
	if errDial != nil {
		fmt.Println("Couldn't dial test chat client", errDial)
		return nil, errDial
	}

	//INITIALIZING GLOBAL VARIABLES
	Rooms = list.New()										//list of Chatroom objects
	Users = make(map[string] *User) 						//list of UserObjects
	PaxosServerConnections = make(map[int] *rpc.Client)		//map of all connections
	ClientConn = chatConn
	ChanMessages = make(chan *multipaxos.SendMessageArgs)

	//GETTING ALL PAXOS SERVER CONNECTIONS
	getServerArgs :=  &multipaxos.GetServersArgs{}
	getServerReply := &multipaxos.GetServersReply{}

	//get servers
	err := chatclient.GetServers(getServerArgs, getServerReply)
	if err != nil {
		fmt.Println("Error occured while getting servers", err)
		return nil, err
	}

	PaxosServers = getServerReply.Servers
	fmt.Println("Set the Paxos Servers", len(PaxosServers))

	for i := 0; i < len(PaxosServers); i++ {
		fmt.Println("Adding rpc connections for", PaxosServers[i])
		currPort := PaxosServers[i]

		serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(currPort))
		if dialErr != nil {
			fmt.Println("Error occurred while dialing to all servers", dialErr)
			return nil, dialErr
		} else {
			PaxosServerConnections[currPort] = serverConn
		}
	}

	http.HandleFunc("/css/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL.Path[5:])
			f, err := os.Open(r.URL.Path[5:])

			if err != nil {
				fmt.Println(err)
			}

			http.ServeContent(w, r, ".css", time.Now(), f)
		})
	http.HandleFunc("/js/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL.Path[4:])
			f, err := os.Open(r.URL.Path[4:])

			if err != nil {
				fmt.Println(err)
			}

			http.ServeContent(w, r, ".js", time.Now(), f)
		})
	http.HandleFunc("/images/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL.Path[8:])
			f, err := os.Open(r.URL.Path[8:])

			if err != nil {
				fmt.Println(err)
			}

			http.ServeContent(w, r, ".jpg", time.Now(), f)
		})

	http.HandleFunc("/", startPageHandler)
	http.Handle("/join", websocket.Handler(NewUser))

	//Finished creating new chat client successfully
	return chatclient, nil
}

func startPageHandler(w http.ResponseWriter, r *http.Request){
	t, _ := template.ParseFiles("startPage.html")
	if t == nil {
		fmt.Println("t is nil")
	}
	if w == nil {
		fmt.Println("w is nil?")
	}
	t.Execute(w, nil)
}

func (cc *ChatClient) FailedMessages() {
	for {
		select {
		case req := <-ChanMessages:
			randomSeconds := rand.Int() % 3
			time.Sleep(time.Second*time.Duration(randomSeconds))
			cc.SendMessage(req, &multipaxos.SendMessageReplyArgs{})
		}
	}
}

func NewUser(ws *websocket.Conn) {
	username := ws.Request().URL.Query().Get("username")
	fmt.Println("Username is", username)

	if username == "" {
		err := errors.New("invalid input for user")
		marshalled, err := json.Marshal(err)
		ws.Write(marshalled)
		fmt.Println(err)
		return
	}

	joiningUser := &User{
		Connection : ws,
		Name : username,
		Rooms : list.New(),
		TimeRecd : time.Now(),
		FirstTime: true,
	}

	if _, ok := Users[username]; ok {
	}

	Users[username] = joiningUser
	cc := &ChatClient{}

	go joiningUser.GetInfoFromUser(ws)
	go cc.FailedMessages()
	joiningUser.SendMessagesToUser()

}

func (user *User) GetInfoFromUser (ws *websocket.Conn) {
	for {
		var content string
		err := websocket.Message.Receive(user.Connection, &content)

		if err != nil {
			fmt.Println("err while receiving a message!", err)
			return
		}

		msg := ChatMessage {
			User: user.Name,
			Room:  "roomName",
			Content: content,
			Timestamp:  time.Now(),
		}
		msgString, err := msg.ToString()
		if err!= nil {
			fmt.Println("Why couldn't message become a string.. unlikely", msgString , err)
		}
		fmt.Println(msgString)

		randPort := PaxosServers[rand.Int()%len(PaxosServers)]

		bytes, marshalErr := json.Marshal(msg)
		if marshalErr != nil {
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
	for {
		time.Sleep(time.Second*3)
		randPort := PaxosServers[rand.Int()%len(PaxosServers)]
		conn := PaxosServerConnections[randPort]
		args := &multipaxos.CommitReplyArgs{}
		reply := &multipaxos.FileReply{}

		errCall := conn.Call("PaxosServer.ServeMessageFile", &args, &reply)

		if(errCall != nil){
			fmt.Println(errCall)
			return errCall
		}

		var err error

		if len(reply.File) <= 0 {
			continue
		}

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

			//If the user just logged in display all previous messages so it catches up
			if user.FirstTime {
				user.Connection.Write(bytes.Trim(msgBytes, "\x00"))
				user.TimeRecd = messageTime
			} else {
				if(messageTime.After(user.TimeRecd)){
					user.Connection.Write(bytes.Trim(msgBytes, "\x00"))
					user.TimeRecd = messageTime
				}
			}
		}
		fmt.Println(err)
		user.FirstTime = false
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
		return nil, err
	}
	return &Page{
		Title: title,
		Body: body,
	}, nil
}

func (cc *ChatClient)SendMessage(args *multipaxos.SendMessageArgs, reply *multipaxos.SendMessageReplyArgs) error {
	go func() {
		conn := PaxosServerConnections[args.PaxosPort]

		errCall := conn.Call("PaxosServer.SendMessage", &args, &reply)
		if errCall != nil {
			fmt.Println(args.PaxosPort, errCall)
			ChanMessages <- args
		} else {
			ChanCommitedMessages <- args
		}
	}()

	return nil
}

func (cc *ChatClient) SendReplaceMessage(args *multipaxos.ReplaceServerArgs, reply *multipaxos.ReplaceServerReply) error {
	conn := PaxosServerConnections[args.PaxosNode]
	errCall := conn.Call("PaxosServer.ReplaceServer", &args, &reply)
	fmt.Println(args.PaxosNode, errCall)

	return errCall
}

func (cc *ChatClient)GetServers(args *multipaxos.GetServersArgs, reply*multipaxos.GetServersReply) error {
	return ClientConn.Call("PaxosServer.GetServers", args, reply)
}

func (cc *ChatClient) GetLogFile(args *multipaxos.FileArgs, reply *multipaxos.FileReply) error{
	conn := PaxosServerConnections[args.Port]
	err := conn.Call("PaxosServer.ServeMessageFile", args, reply)
	if(err != nil){
		fmt.Println(err)
		return err
	}
	return nil
}

func (cc *ChatClient) UpdateServers(deadNode, newNode int) error {
	getServerArgs :=  &multipaxos.GetServersArgs{}
	getServerReply := &multipaxos.GetServersReply{}

	err := cc.GetServers(getServerArgs, getServerReply)
	if err != nil {
		return err
	}
	PaxosServers = getServerReply.Servers

	//delete connection to deadnode and update this one
	delete(PaxosServerConnections, deadNode)

	serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(newNode))
	if dialErr != nil {
		return  dialErr
	} else {
		PaxosServerConnections[newNode] = serverConn
	}

	return nil
}

func(cc *ChatClient) SendManualRecover(args *multipaxos.ManualRecoverArgs, reply *multipaxos.ManualRecoverReply) error {
	conn := PaxosServerConnections[args.PaxosNode]
	err := conn.Call("PaxosServer.SendRecover", args, reply)
	return err
}

