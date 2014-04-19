package logger


import (
	"errors"
	"time"
)


type LogMessage struct {
	timestamp time.Time
	proposalID int
	value 	   []byte
}

type Logger struct{
	fileName string
}

func NewLogger() (*Logger, error) {
	//TODO

	return nil, errors.New("Unimplemented")
}

func (log *Logger) post() error {
	//TODO
	return nil
}

func (log *Logger) getLog() error {
	//TODO
	return nil
}

//TODO idk what else this file should have.. 



