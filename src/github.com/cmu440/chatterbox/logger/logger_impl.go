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
	file *os.File
}

func NewLogger() (*Logger, error) {
	file, errCreate := os.Create("P3-IMPL-LOGGER")

	if errCreate != nil {
		return nil, errCreate
	}

	log := &Logger {
		file : file,
	}

	return log, nil
}

/**
	post a new log message
 */
func (log *Logger) Post(msg []byte) error {
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

func (log *Logger) GetLog() (*os.File, error) {
	return log.file, nil
}

/* Close the logging file after finishing
 */
func (log *Logger) CloseLog() error {
	errClose := log.file.Close()
	return errClose
}
