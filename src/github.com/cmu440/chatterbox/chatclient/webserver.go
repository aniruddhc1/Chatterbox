package chatclient
/*
import (
	"github.com/cmu440/chatterbox/handlers"
	"net/http"
	"code.google.com/p/go.net/websocket"
	"github.com/cmu440/chatterbox/chatclient"
	"mango"

)

func main(){
	layout, render := handlers.LayoutAndRender()
	stack := new(Stack)
	stack.Middleware(layout, render)

	http.Handle("/chat", websocket.Handler(chatclient.StartConnection))
	http.HandleFunc("/join", stack.HandlerFunc(handlers.Join))
	http.HandleFunc("/", stack.HandlerFunc(handlers.Home))
	http.HandleFunc("/public/", handleAssets)

	go chatclient.NewChatClient()

	err := http.ListenAndServe(":8080", nil)
	if err != nil{
		panic("listen and serve : ", err)
	}
}

func handleAssets(writer http.ResponseWriter, request *http.Request){
	http.ServeFile(writer, request, request.URL.Path[len("/"):])
}
*/
