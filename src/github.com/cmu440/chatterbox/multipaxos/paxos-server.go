package multipaxos

/* TODO Things left to do for paxos implentation
1. Write the function SendMessage (basically a wrapper around paxos) which gets the message
   from the chat client and receives an error if any step of the paxos process fails.
   If it fails make it start the paxos again
2. Do the exponential backoff stuff to prevent livelock
3. Change recovery from logs to a complete file based log system
4. Write a lot more tests,
5. Change testing to not only just killing but also like sleeping to make a server fall behind
6.
*/

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"time"
	"sync"
)

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

type paxosServer struct {
	Port           int
	MasterHostPort string
	NumNodes       int
	NumConnected   int                 //number of servers that are currently connected and active
	RPCConnections map[int]*rpc.Client //from port number to the rpc connection so we don't have to dial everytime
	Servers        []int               //port numbers of all the servers in the ring
	listener       net.Listener
	RoundID        int
	RoundIDLock 	*sync.Mutex

	//Required for proposer role
	ProposalID           int
	MaxSeenProposalID 	 int
	MsgQueue             *list.List
	ProposeAcceptedQueue *list.List //contains the ProposeReplyArgs received from all the acceptors who have accepted the proposal; Reset at each round
	AcceptedQueue        *list.List //contains the AcceptReplyArgs received rom all acceptors who have accepted the Accept request; reset after every round

	//Required for accepter role
	ToCommitQueue *list.List //list of proposalID-data keyvaluepair of accepted but yet not committed messages
	MaxPromisedID int

	//Required for learner role
	CommittedMsgs     map[int][]byte //message committed for every round of Paxos
	CommittedMsgsFile *os.File

	RegisterLock *sync.Mutex


}

func NewPaxosServer(masterHostPort string, numNodes, port int) (*paxosServer, error) {


	timeString := time.Now().String()
	file, err := os.Create(strconv.Itoa(port)+"|"+ timeString)

	if (err != nil){
		fmt.Println(err)
		return nil, err
	}

	//Initialize paxos server
	paxosServer := &paxosServer{
		Port:           port,
		MasterHostPort: masterHostPort,
		NumNodes:       numNodes,
		NumConnected:   0,
		RPCConnections: make(map[int]*rpc.Client),
		Servers:        make([]int, numNodes),
		RoundID:        1,
		RoundIDLock: 	&sync.Mutex{},

		//Required for proposal role
		ProposalID:           0,
		MaxSeenProposalID:	  0,
		MsgQueue:             list.New(),
		ProposeAcceptedQueue: list.New(),
		AcceptedQueue:        list.New(),

		//Required for acceptor role
		ToCommitQueue: list.New(),
		MaxPromisedID: 0,

		//Required for learner role
		CommittedMsgs:     make(map[int][]byte),
		CommittedMsgsFile: file,

		RegisterLock: &sync.Mutex{},
	}

	//Register the server http://angusmacdonald.me/writing/paxos-by-example/to RPC
	errRegister := rpc.RegisterName("PaxosServer", Wrap(paxosServer))
	if errRegister != nil {
		fmt.Println("An error occured while doing rpc register", errRegister)
		return nil, errRegister
	}
	rpc.HandleHTTP()

	//Start a lister for the server
	listener, errListen := net.Listen("tcp", ":"+strconv.Itoa(paxosServer.Port))
	if errListen != nil {
		fmt.Println("An error occured while trying to listen", errListen)
		return nil, errListen
	}

	paxosServer.listener = listener

	go http.Serve(listener, nil)

	//If the server is a slave create a connection to the master
	var regular *rpc.Client
	var errDial error
	if masterHostPort != "" {
		regular, errDial = rpc.DialHTTP("tcp", masterHostPort)
		if errDial != nil {
			fmt.Println("Slave couldn't connect to master", errDial)
			return nil, errDial
		}
	}

	//Wait till all paxos servers in group have joined
	for {
		fmt.Println("TRYING", paxosServer.Port)
		args := &RegisterArgs{paxosServer.Port}
		reply := &RegisterReplyArgs{}
		var err error

		if masterHostPort == "" {
			//This is a master paxos server that others will register to
			err = paxosServer.RegisterServer(args, reply)
			if err != nil {
				fmt.Println(paxosServer.Port, err, "SLEEPING for SECOND")
				time.Sleep(time.Second)
				continue
			}
		} else {
			//This is a regular paxos server
			err := regular.Call("PaxosServer.RegisterServer", args, reply)
			if err != nil {
				fmt.Println(paxosServer.Port, err, "SLEEPING for SECOND")
				time.Sleep(time.Second)
				continue
			}
		}

		fmt.Println("Setting paxos servers", paxosServer.Port, reply.NumConnected)
		paxosServer.Servers = reply.Servers
		break
	}

	err = paxosServer.CreatePaxosConnections()
	if err!= nil {
		fmt.Println(err)
		return nil, err
	}

	return paxosServer, nil
}

func (ps *paxosServer) CreatePaxosConnections() error{
	fmt.Println("Adding rpc connections for", ps.Port)
	//Create rpc connections to all servers
	for i := 0; i < ps.NumNodes; i++ {
		currPort := ps.Servers[i]
		fmt.Println(currPort, ps.Port)
		serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(currPort))
		if dialErr != nil {
			fmt.Println("Error occured while dialing to all servers", dialErr)
		} else {
			ps.RPCConnections[currPort] = serverConn
		}
	}

	if len(ps.RPCConnections) == 0 {
		return errors.New("Couldn't create any rpc connections")
	}

	return nil
}

func (ps *paxosServer) RegisterServer(args *RegisterArgs, reply *RegisterReplyArgs) error {
	ps.RegisterLock.Lock()
	alreadyJoined := false

	for i := 0; i < ps.NumConnected; i++ {
		if ps.Servers[i] == args.Port {
			alreadyJoined = true
		}
	}


	if !alreadyJoined {
		fmt.Println("IN REGISTER", ps.NumConnected, args.Port)
		ps.Servers[ps.NumConnected] = args.Port
		ps.NumConnected++
	}

	reply.Servers = ps.Servers
	reply.NumConnected = ps.NumConnected

	if ps.NumConnected != ps.NumNodes {
		ps.RegisterLock.Unlock()
		return errors.New("Not all servers have joined")
	}

	ps.RegisterLock.Unlock()
	return nil
}

type FileReply struct{
	File *os.File
}

func (ps *paxosServer) ServeMessageFile(args *CommitReplyArgs, reply *FileReply) error{
	reply.File = ps.CommittedMsgsFile

	return nil
}

func (ps *paxosServer) SendMessage(args *SendMessageArgs, reply *SendMessageReplyArgs) error {
	fmt.Println("in send message")
	ps.MsgQueue.PushBack(args.Value)
	return ps.Propose(args, reply)
}

//Functions related to Testing
func (ps *paxosServer) CheckKill(tester *Tester, currStage string, currTime string) error {
	//  sendPropose, sendAccept, sendCommit, receivePropose
	//  receiveAccept, receiveCommit
	//	killTime string //start, mid, end

	var err error

	if tester.Stage == currStage && tester.Time == currTime {
		fmt.Println("Killing", ps.Port, "Need to stop at", tester.Stage, tester.Time,
			"Stopping at", currStage, currTime)
		ps.listener.Close()
		for _, conn := range ps.RPCConnections {
			 err = conn.Close()
		}
	}

	return err
}


//Functions related to Recovery
func (ps *paxosServer) SendRecover() error {
	logs := make(map[int]*RecoverReplyArgs)
	maxRound := 0

	for port, conn := range ps.RPCConnections {
		args := &RecoverArgs{ps.RoundID}
		reply := &RecoverReplyArgs{}

		err := conn.Call("PaxosServer.HandleRecover", args, reply)

		if err != nil {
			fmt.Println("Error occured while calling Handle Recover", err)
			return err
		}

		logs[port] = reply
		if maxRound < reply.RoundID {
			maxRound = reply.RoundID
		}
	}

	//check that all logs are consistent and update this servers logs
	for i:=ps.RoundID; i<maxRound; i++ {
		//check to make sure that all of them are the same for that round id
		var val []byte
		for port, log := range logs {
			if log.RoundID < i {
				if val == nil {

					val = log.CommittedValues[i]
				}
				if bytes.Equal(log.CommittedValues[i], val) {
					//This should never happen
					return errors.New("Paxos server " + strconv.Itoa(port) + " logs are not correct")
				}
			}
		}

		ps.CommittedMsgs[ps.RoundID] = val
		fmt.Println(ps.RoundID, ps.Port, ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		ps.RoundID++
	}

	return nil
}

func (ps *paxosServer) HandleRecover(args *RecoverArgs, reply *RecoverReplyArgs) error {
	//If this server is behind the server trying to recover then first recover yourself 
	if args.RoundID < ps.RoundID {
		err := ps.SendRecover()
		if err != nil {
			errors.New("Couldn't recover this one so cant send updated logs")
		}
	}

	reply.RoundID = ps.RoundID
	reply.CommittedValues = ps.CommittedMsgs
	return nil
}

//Functions related to Proposer
func (ps *paxosServer) Propose(args *SendMessageArgs, _ *SendMessageReplyArgs) error {
	ps.CheckKill(&args.Tester, "sendPropose", "start")

	if ps.MaxSeenProposalID > ps.ProposalID {
		ps.ProposalID = ps.MaxSeenProposalID + 1
	} else {
		ps.ProposalID++
	}

	majority := ps.NumNodes/2 + ps.NumNodes%2
	maxPair := &KeyValuePair{-1, nil}

	if len(ps.RPCConnections) == 0{
		err := ps.CreatePaxosConnections()
		if err == nil {
			fmt.Println("Error Occured", err)
		}
	}

	for _, conn := range ps.RPCConnections  {
		proposeArgs := &ProposeArgs{
			RoundID:    ps.RoundID,
			ProposalID: ps.ProposalID,
			Proposer:   ps.Port,
		}
		proposeReply := &ProposeReplyArgs{}

		err := conn.Call("PaxosServer.HandleProposeRequest", proposeArgs, proposeReply)
		if err != nil {
			fmt.Println("Error while calling HandleProposeRequest", err)
		}

		fmt.Println("Proposing reply port", proposeReply.AcceptorPort, "round ", proposeReply.RoundID, "accepted? ",
			proposeReply.Accepted, "soumya is ", proposeReply.Pair)

		if time.Now().UnixNano()%2 == 0 {
			ps.CheckKill(&args.Tester, "sendPropose", "mid")
		}

		if proposeReply.Pair != nil {
			if proposeReply.Pair.ProposalID > ps.MaxSeenProposalID {
				ps.MaxSeenProposalID = proposeReply.Pair.ProposalID
			}
		}

		if proposeReply.RoundID != ps.RoundID {
			fmt.Println("Propose Round Id's do not match need to recover my round id:",ps.RoundID, "proposeReply id", proposeReply.RoundID)
			err := ps.SendRecover()
			if err != nil {
				fmt.Println("Couldn't recover", err)
			}
		} else if proposeReply.Accepted {
			fmt.Println(proposeReply.AcceptorPort, "Accepted!")
			ps.ProposeAcceptedQueue.PushBack(proposeReply)
			if proposeReply.Pair != nil && proposeReply.Pair.ProposalID > maxPair.ProposalID {
				maxPair.ProposalID = proposeReply.Pair.ProposalID
				maxPair.Value = proposeReply.Pair.Value
			}
		} else {
			continue
		}
	}

	ps.CheckKill(&args.Tester, "sendPropose", "end")
	fmt.Println("The majority is", majority, "Num accepted is", ps.ProposeAcceptedQueue.Len())

	if ps.ProposeAcceptedQueue.Len() >= majority {
		if maxPair.ProposalID != -1 {
			fmt.Println("maxpair not nil, sending accept request  ", maxPair.ProposalID)
			return ps.SendAcceptRequests(ps.ProposeAcceptedQueue, maxPair.ProposalID, maxPair.Value, &args.Tester)
		} else {
			return ps.SendAcceptRequests(ps.ProposeAcceptedQueue, ps.ProposalID, args.Value, &args.Tester)
		}
	}

	return errors.New("Couldn't get a majority to accept request to proposal")
}

func (ps *paxosServer) SendAcceptRequests(acceptors *list.List, id int, value []byte, tester *Tester) error {
	fmt.Println("Sending Accept Requests")
	ps.CheckKill(tester, "sendAccept", "start")
	majority := ps.NumNodes/2 + ps.NumNodes%2

	for e := acceptors.Front(); e != nil; e = e.Next() {
		reply := e.Value.(*ProposeReplyArgs)
		fmt.Println(reply.AcceptorPort)
		conn := ps.RPCConnections[reply.AcceptorPort]

		acceptArgs := &AcceptRequestArgs{
			RoundID:    ps.RoundID,
			ProposalID: id,
			Value:      value,
		}
		acceptReply := &AcceptReplyArgs{}

		if time.Now().UnixNano()%2 == 0 {
			ps.CheckKill(tester, "sendAccept", "mid")
		}

		err := conn.Call("PaxosServer.HandleAcceptRequest", acceptArgs, acceptReply)
		if err != nil {
			fmt.Println("error in send accept request ", err)
		}

		if acceptReply.Accepted {
			fmt.Println(reply.AcceptorPort, "Paxos server accepted")
			ps.AcceptedQueue.PushBack(acceptReply)
		}
	}

	_ = ps.CheckKill(tester, "sendAccept", "end")

	if ps.AcceptedQueue.Len() >= majority {
		err := ps.SendCommit(acceptors, value, tester)
		return err
	}
	return errors.New("Couldn't reach a majority to send the accept requests")
}

func (ps *paxosServer) SendCommit(acceptors *list.List, value []byte, tester *Tester) error {
	ps.CheckKill(tester, "sendCommit", "start")
	fmt.Println("Send Commit")
	ImAnAcceptor := false

	for e := acceptors.Front(); e != nil; e = e.Next() {
		reply := e.Value.(*ProposeReplyArgs)

		if reply.AcceptorPort == ps.Port {
			ImAnAcceptor = true
			continue
		}

		conn := ps.RPCConnections[reply.AcceptorPort]

		if time.Now().UnixNano()%2 == 0 {
			ps.CheckKill(tester, "sendCommit", "mid")
		}

		commitArgs := &CommitArgs{Value: value, RoundID: ps.RoundID}
		commitReply := &CommitReplyArgs{}

		err := conn.Call("PaxosServer.HandleCommit", commitArgs, commitReply)

		if err != nil {
			fmt.Println("error in send commit request ", err)
		}
	}

	if ImAnAcceptor {
		commitArgs := &CommitArgs{Value: value, RoundID: ps.RoundID}
		commitReply := &CommitReplyArgs{}
		conn := ps.RPCConnections[ps.Port]
		err := conn.Call("PaxosServer.HandleCommit", commitArgs, commitReply)
		if err != nil {
			fmt.Println("error in send commit request ", err)
		}
	}

	ps.CheckKill(tester, "sendCommit", "end")

	return nil
}

//Functions related to Acceptor
func (ps *paxosServer) HandleProposeRequest(args *ProposeArgs, reply *ProposeReplyArgs) error {
	fmt.Println("Handle Propose Request")
	reply.RoundID = ps.RoundID
	reply.AcceptorPort = ps.Port

	if args.ProposalID > ps.MaxSeenProposalID {
		ps.MaxSeenProposalID = args.ProposalID
	}

	if args.RoundID < ps.RoundID {
		fmt.Println("Can't accept propose because rounds mismatch")
		reply.Accepted = false
		return nil
	} else if args.RoundID > ps.RoundID {
		err := ps.SendRecover()
		if err != nil {
			fmt.Println("Paxos Server", ps.Port, "was behind and couldn't recover properly")
			reply.Accepted = false
			return nil
		}
	} else if ps.MaxPromisedID >= args.ProposalID {
		fmt.Println("Couldn't accept propose because max proposed is higher than proposal id", ps.MaxPromisedID, args.ProposalID)
		reply.Accepted = false
		return nil
	}
	reply.Accepted = true
	if ps.ToCommitQueue.Len() > 0 && ps.ToCommitQueue.Front() != nil {
		commitMsg := ps.ToCommitQueue.Front().Value.(KeyValuePair)
		reply.Pair = &KeyValuePair{commitMsg.ProposalID, commitMsg.Value}
	}
	return nil
}

func (ps *paxosServer) HandleAcceptRequest(args *AcceptRequestArgs, reply *AcceptReplyArgs) error {
	fmt.Println("Handle Accept Request")
	reply.RoundID = ps.RoundID
	reply.AcceptorPort = ps.Port

	if args.RoundID < ps.RoundID {
		reply.Accepted = false
		return nil
	} else if args.RoundID > ps.RoundID {
		err := ps.SendRecover()
		if err != nil {
			fmt.Println("Couldn't recover", err)
		}
	} else if ps.MaxPromisedID >= args.ProposalID {
		reply.Accepted = false
		return nil
	}

	ps.ToCommitQueue.PushBack(KeyValuePair{args.ProposalID, args.Value})
	reply.Accepted = true
	return nil
}

func (ps *paxosServer) HandleCommit(args *CommitArgs, _ *CommitReplyArgs) error {
	fmt.Println("Handle commit message paxos server", ps.Port)

	ps.CommittedMsgs[args.RoundID] = args.Value

	_, err := ps.CommittedMsgsFile.Write(args.Value)
	if(err != nil){
		return err
	}

	if ps.RoundID < args.RoundID {
		err := ps.SendRecover()
		if err != nil {
			fmt.Println("Couldn't recover", err)
		}
	}

	fmt.Println(ps.RoundID, ps.Port, "________________________________________")
	ps.RoundID++

	ps.ProposeAcceptedQueue = list.New()
	ps.AcceptedQueue = list.New()

	for e := ps.ToCommitQueue.Front(); e != nil; e = e.Next() {
		if bytes.Equal(e.Value.(KeyValuePair).Value, args.Value) {
			ps.ToCommitQueue.Remove(e)
			break
		}
	}
	return nil
}

func (ps *paxosServer) GetServers(_ *GetServersArgs, reply *GetServersReply) error {

	if ps.NumConnected == ps.NumNodes {
		reply.Servers = ps.Servers
		fmt.Println("number of nodes connected are ", ps.NumConnected)
		reply.Ready = true
	} else {
		fmt.Println("number of nodes connected are ", ps.NumConnected)
		reply.Ready = false

	}
	return nil
}

