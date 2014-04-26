package chatclient

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"net/rpc"
)

type JoiningUser struct{
	CurrRoom *CurrRoom
	Connection *websocket.Conn
	Name User
	Send chan Message
}

type User struct{
	Name string
}

type CurrRoom struct{
	Users map[string]*JoiningUser
	Broadcast chan Message

}

type Message struct{
	User *User
	Time string
	Contents string
}

func StartChatRoom(){
	activeRoom := &CurrRoom{
		Users : make(map[string]*JoiningUser),
		Broadcast : make(chan Message),
	}
	go activeRoom.Run()
}

func (chatRoom *CurrRoom) Run(){
	for{
		select{
		case msg := <-chatRoom.Broadcast:
			for _, user := chatRoom.Users{
				user.Send <- msg
			}
		//close case
		}
	}
}

var activeRoom *CurrRoom = &CurrRoom{}

func StartConnection(ws *websocket.Conn){
	username := ws.Request().URL.Query().Get("username")

	if username == "" {
		return
	}

	joiningUser := &JoiningUser{
		CurrRoom: activeRoom,
		Connection : ws,
		Name : username,
	}

	activeRoom.Users[username] = joiningUser

	go joiningUser.ToClient()

	joiningUser.FromClient()

	//stop chatroom

}

func (user *JoiningUser) ToClient(){

	//simple rpc call to sendMessage in paxos server


}

func (user *JoiningUser) FromClient(){

	//make rpc call to paxos server, read the file here

	for{
		var message string

		if err != nil{
			fmt.Println(err)
		}


	}
}
















