package paxos

import(
	"testing"
	"fmt"
)


func TestStartServer(t *testing.T){
	_, err := NewPaxosServer("", 2, 8080) //starting master server

	_, err1 := NewPaxosServer("localhost:8080", 2, 9999) // starting node

	fmt.Println(err)
	fmt.Println(err1)


}
