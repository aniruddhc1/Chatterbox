package logger


import (
	"time"
	"os"
	"encoding/json"
	"github.com/cmu440/chatterbox/chatclient"
)


type LogMessage struct {
	timestamp time.Time
	proposalID int
	value 	   []byte
}

type Logger struct{
	file *File
}

func NewLogger() (*Logger, error) {
	//TODO

	file, errCreate := os.Create("P3-IMPL-LOGGER")

	if errCreate != nil {
		return nil, errCreate
	}

	log := &Logger {
		file : file,
	}

	return log, nil
}

/*
 *
 */
func (log *Logger) Post(msg []byte) error {
	//TODO

	chatMsg := chatclient.ChatMessage{}

	errUnmarshal := json.Unmarshal(msg, chatMsg)

	if errUnmarshal != nil {
		return errUnmarshal
	}
	msgString, errString := chatMsg.ToString()

	if errString != nil {
		return errString
	}

	_, errWrite := log.file.WriteString(msgString)

	if errWrite != nil {
		return errWrite
	}

	return nil
}

/*

 */
func (log *Logger) GetLog() (*File, error) {
	//TODO
	return log.file, nil
}

/* Close the logging file after finishing
 */
func (log *Logger) CloseLog() error {
	errClose := log.file.Close()
	return errClose
}

//TODO idk what else this file should have.. 



