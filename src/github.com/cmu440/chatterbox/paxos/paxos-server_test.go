package paxos

import(
	"testing"
	"net/rpc"
	"strconv"
	"fmt"
)

func TestStart2(t *testing.T){
	fmt.Println("entering")
	_, dialErr := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(8080))
	fmt.Println(dialErr)

	_, dialErr1 := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(9999))
	fmt.Println(dialErr1)

	if dialErr1 != nil{
		t.Error("hi")
	}


}
