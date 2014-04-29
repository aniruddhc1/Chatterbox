package chatclient

import (
	"time"
	"errors"
)

type ChatMessage struct {
	User string
	Room  string
	Content  string
	Timestamp  time.Time
}


func (msg *ChatMessage) ToString() (string, error) {
	if msg.User == "" {
		return "", errors.New("Message doesn't have user")
	}

	if msg.Room == "" {
		return "", errors.New("Message doesn't have room")
	}

	if msg.Content == "" {
		return "", errors.New("Message doesn't have content")
	}

	msgString := msg.User + "|" + msg.Room + "|" + msg.Content + "|" + msg.Timestamp.Format(time.RFC3339Nano) + "\n"

	return msgString, nil
}
