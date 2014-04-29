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
	"io/ioutil"
	"html/template"
	"os"
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
	http.HandleFunc("/chat/", chatHandler)
	http.Handle("/join", websocket.Handler(NewUser))

	fmt.Println("Finished Creating New Chat Client")
	return chatclient, nil
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in index.html handler")
	fmt.Println(r.Method)
	if r.Method == "GET" {
		fmt.Println("method was get")
		err := r.ParseForm()
		fmt.Println(r.URL)

		if err != nil {
			fmt.Println("couldn't parse form", err)
		}

		fmt.Println(r.Form)
	}


	fmt.Println(r.Form)
	fmt.Println("Username in chat handler", r.PostForm, r.RequestURI)

	t, _ := template.ParseFiles("index.html")
	if t == nil {
		fmt.Println("why is t nil?")
	}
	if w == nil {
		fmt.Println("why is w nil?")
	}

	t.Execute(w, nil)
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

	Users[username] = joiningUser

	bytes, _ := json.Marshal("HI")
	fmt.Println("Writing a message")
	ws.Write(bytes)

	go joiningUser.GetInfoFromUser(ws)
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

		var room string // parse content to get room
		var msgContent string //get the actual message

		msg := ChatMessage {
			User: user.Name,
			Room:  room,
			Content: msgContent,
			Timestamp:  time.Now(),
		}

		msgString, err := msg.ToString()
		fmt.Println(msgString)
		//TODO  //DO the paxos thing and go from there

	}
}

func (user *User) SendMessagesToUser() error{
	fmt.Println("Inside SENDMESSAGESTOUSER")
	for {
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
		fmt.Println(args.PaxosPort, errCall)
	}()

	return nil
}

func (cc *ChatClient)GetServers(args *multipaxos.GetServersArgs, reply*multipaxos.GetServersReply) error {
	return ClientConn.Call("PaxosServer.GetServers", &args, &reply)
}


func (cc *ChatClient) GetLogFile(args *multipaxos.FileArgs, reply *multipaxos.FileReply) error{
	conn := PaxosServerConnections[args.Port]
	err := conn.Call("PaxosServer.ServeMessageFile", &args, &reply)

	if(err != nil){
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
