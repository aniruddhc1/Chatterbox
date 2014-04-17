package paxos

import "errors"


type MessageType int

const (
	ProposeRequest = iota + 1
	ProposeResponse
	AcceptRequest
	AcceptResponse
	Commit
)

type ProposeRequestArgs struct {
	proposalID int
}

type ProposeResponseArgs struct {
	acceptPropose bool
	previosProposalId int
	previousValue []byte
}

type AcceptRequestArgs struct {
	proposalId int
	value []byte
}

type AcceptResponseArgs struct {
	accepted bool
}

type CommitArgs struct {
	value []byte
}

type PaxosMessage struct {
	msgType     MessageType
	proposalID  int
	value       []byte
}

type PaxosServer struct {
	//TODO
}

type RegisterArgs struct {
	//TODO
}

type RegisterReply struct {
	//TODO
}

type GetServersArgs struct {
	//TODO
}

type GetServersReply struct {
	//TODO
}

func NewPaxosServer() (PaxosServer, error) {
	//TODO
	return nil
}

func (ps *PaxosServer) RegisterServer(args *RegisterArgs, reply *RegisterReply) error {
	//TODO
	return nil
}

func (ps *PaxosServer) GetServers(args GetServersArgs, reply GetServersReply) error {

}

func (ps *PaxosServer) ProposeRequest() error{
	//TODO
	return errors.New("not implemented")
}

func (ps *PaxosServer) HandleProposeRequest(msg PaxosMessage) error{
	//TODO
	return errors.New("not implemented")
}

func (ps *PaxosServer) HandleProposeResponse(msg PaxosMessage) error {
	//TODO
	return errors.New("not implemented")
}

func (ps *PaxosServer) HandleAcceptRequest(msg PaxosMessage) error{
	//TODO
	return errors.New("not implemented")
}

func (ps *PaxosServer) HandleAcceptResponse(msg PaxosMessage) error{
	//TODO
	return errors.New("not implemented")
}

func (ps *PaxosServer) HandleAccept(msg PaxosMessage) error{
	//TODO
	return errors.New("not implemented")
}
