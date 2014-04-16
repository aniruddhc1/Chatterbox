package paxos

import "errors"


type MessageType int

const (
	ProposeRequest = iota + 1
	ProposeResponse
	ProposeReject
	AcceptRequest
	AcceptResponse
	AcceptReject
)

type PaxosMessage struct {
	msgType     MessageType
	proposalID  int
	value       []byte
}


func ProposeRequest() error{
	return errors.New("not implemented")
}

func HandleProposeRequest(msg PaxosMessage) error{
	return errors.New("not implemented")
}

func HandleProposeResponse(msg PaxosMessage) error {
	return errors.New("not implemented")
}

func HandleAcceptRequest(msg PaxosMessage) error{
	return errors.New("not implemented")
}

func HandleAcceptResponse(msg PaxosMessage) error{
	return errors.New("not implemented")
}

func HandleAccept(msg PaxosMessage) error{
	return errors.New("not implemented")
}
