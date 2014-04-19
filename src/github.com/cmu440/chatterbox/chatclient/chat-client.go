package chatclient

import (
	"errors"
)



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
