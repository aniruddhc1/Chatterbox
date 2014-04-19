package chatclient

import (
	"time"
	"errors"
)

type ChatMessage struct {
	user string
	room  string
	content  string
	timestamp  time.Time
}


func (msg *ChatMessage) ToString() (string, error) {
	if msg.user == "" {
		return "", errors.New("Message doesn't have user")
	}

	if msg.room == "" {
		return "", errors.New("Message doesn't have user")
	}

	if msg.content == "" {
		return "", errors.New("Message doesn't have content")
	}

	msgString := msg.user + "|" + msg.room + "|" + msg.content + "|" + msg.timestamp.Format(time.RFC3339Nano) + "\n"

	return msgString, nil
}
