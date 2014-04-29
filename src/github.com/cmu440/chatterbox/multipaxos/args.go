package multipaxos

type KeyValuePair struct {
	ProposalID int
	Value      []byte
}

type GetServersArgs struct {
	//intentionally left empty
}

type GetServersReply struct {
	Ready   bool
	Servers []int
}

type SendMessageArgs struct {
	Value  []byte
	Tester Tester
	PaxosPort int
}

type Tester struct {
	Stage string //sendPropose, sendAccept, sendCommit, receivePropose, receiveAccept, receiveCommit
	Time string //start, mid, end
	Kill bool //true means kill else sleep to delay the server
	SleepTime int 	//time to slep if above is false
}

type SendMessageReplyArgs struct {
	//this page intentionally left blank.
}

type ProposeArgs struct {
	RoundID    int
	ProposalID int
	Proposer   int //port of the proposer
}

type ProposeReplyArgs struct {
	RoundID      int
	Pair         *KeyValuePair //consists of proposal id and value
	Accepted     bool
	AcceptorPort int
}

type AcceptRequestArgs struct {
	RoundID    int
	ProposalID int
	Value      []byte
}

type AcceptReplyArgs struct {
	Accepted     bool
	AcceptorPort int
	RoundID      int
}

type CommitArgs struct {
	Value   []byte
	RoundID int
}

type CommitReplyArgs struct {
	//intentionally left blank
}

type RecoverArgs struct {
	RoundID int
}

type RecoverReplyArgs struct {
	RoundID int
	CommittedValues map[int] []byte
}

type RegisterArgs struct {
	Port int
}

type RegisterReplyArgs struct {
	Servers []int
	NumConnected int
}

//Will be used by chat client
type GetCommitMsgsArgs struct {
	//TODO
}

type GetCommitMsgsReply struct {
	//TODO
}

type FileReply struct{
	File []byte
}

type FileArgs struct{
	Port int
}

type WakeupRequestArgs struct{

}

type WakeupReplyArgs struct{

}
