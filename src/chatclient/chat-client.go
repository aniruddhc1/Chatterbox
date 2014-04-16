package chatclient

import (
	"time"
	"errors"
)

type chatMessage struct {
	user string
	room  string
	content  string
	timestamp  time.Time
}

type chatClient struct {

}

func NewChatClient(hostport string) (*chatClient, error) {
	return nil, errors.New("Not Implemented")
}

func (*chatClient) CreateNewUser(name string) error {
	return errors.New("Not Implemented")
}

func (*chatClient) JoinChatRoom(room string) error {
	return errors.New("Not Implemented")
}

func (*chatClient) SendMessage(room string, msg string) error {
	return errors.New("Not Implemented")
}
