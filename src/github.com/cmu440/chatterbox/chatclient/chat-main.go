package main

import (
	"net/http"
	"code.google.com/p/go.net/websocket"
	"github.com/cmu440/chatterbox/chatclient"
	"fmt"
)


func main() {
	cc, err := chatclient.NewChatClient("1050", 8080)

	if err != nil {
		fmt.Println("Couldnt start a new chat client - Fail")
		return
	}

	http.Handle("/join", websocket.Handler(cc.NewUser()))
}
