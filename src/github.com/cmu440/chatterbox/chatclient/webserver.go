package chatclient

import (
	"fmt"
	"net/http"
)

func main(){
	_, err := NewChatClient("localhost:8080", 9990)

	if err != nil{
		fmt.Println(err)
	}

	go http.Handle("/", http.FileServer(http.Dir("templates")))

}
