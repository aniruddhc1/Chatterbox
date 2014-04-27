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
	KillStage string //sendPropose, sendAccept, sendCommit, receivePropose, receiveAccept, receiveCommit
	KillTime string //start, mid, end
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
	RoundID        int
	MasterHostPort string
	NumNodes       int
	NumConnected   int                 //number of servers that are currently connected and active
	RPCConnections map[int]*rpc.Client //from port number to the rpc connection so we don't have to dial everytime
	Servers        []int               //port numbers of all the servers in the ring
	listener       net.Listener

	//Required for proposer role
	ProposalID           int
	MsgQueue             *list.List
	ProposeAcceptedQueue *list.List //contains the ProposeReplyArgs received from all the acceptors who have accepted the proposal; Reset at each round
	AcceptedQueue        *list.List //contains the AcceptReplyArgs received rom all acceptors who have accepted the Accept request; reset after every round

	//Required for accepter role
	ToCommitQueue *list.List //list of proposalID-data keyvaluepair of accepted but yet not committed messages
	MaxPromisedID int

	//Required for learner role
	CommittedMsgs     map[int][]byte //message committed for every round of Paxos
	CommittedMsgsFile *os.File
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
		RoundID:        0,
		MasterHostPort: masterHostPort,
		NumNodes:       numNodes,
		NumConnected:   0,
		RPCConnections: make(map[int]*rpc.Client),
		Servers:        make([]int, numNodes),

		//Required for proposal role
		ProposalID:           0,
		MsgQueue:             list.New(),
		ProposeAcceptedQueue: list.New(),
		AcceptedQueue:        list.New(),

		//Required for acceptor role
		ToCommitQueue: list.New(),
		MaxPromisedID: 0,

		//Required for learner role
		CommittedMsgs:     make(map[int][]byte),
		CommittedMsgsFile: file,
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
		args := &RegisterArgs{paxosServer.Port}
		reply := &RegisterReplyArgs{}
		var err error

		if masterHostPort == "" {
			//This is a master paxos server that others will register to
			err = paxosServer.RegisterServer(args, reply)
		} else {
			//This is a regular paxos server
			err := regular.Call("PaxosServer.RegisterServer", args, reply)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}
		}

		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
		} else {
			paxosServer.Servers = reply.Servers
			break
		}
	}

	//Create rpc connections to all servers
	for i := 0; i < numNodes; i++ {
		fmt.Println("Adding rpc connections for", paxosServer.Port)
		currPort := paxosServer.Servers[i]

		serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(currPort))
		if dialErr != nil {
			fmt.Println("Error occured while dialing to all servers", dialErr)
			return nil, dialErr
		} else {
			paxosServer.RPCConnections[currPort] = serverConn
		}
	}

	return paxosServer, nil
}

func (ps *paxosServer) RegisterServer(args *RegisterArgs, reply *RegisterReplyArgs) error {
	alreadyJoined := false

	for i := 0; i < ps.NumConnected; i++ {
		if ps.Servers[i] == args.Port {
			alreadyJoined = true
		}
	}

	var err error
	reply.Servers = ps.Servers

	if !alreadyJoined {
		fmt.Println("IN REGISTER", ps.NumConnected, args.Port)
		ps.Servers[ps.NumConnected] = args.Port
		ps.NumConnected++
	}

	if ps.NumConnected != ps.NumNodes {
		err = errors.New("Not all servers have joined")
	}

	return err
}

type FileReply struct{
	file os.File
}

func (ps *paxosServer) ServeMessageFile(args *CommitReplyArgs, reply *FileReply) error{
	reply.file = *ps.CommittedMsgsFile

	return nil
}

func (ps *paxosServer) SendMessage(args *SendMessageArgs, reply *SendMessageReplyArgs) error {
	fmt.Println("in send message")
	ps.MsgQueue.PushBack(args.Value)
	return ps.Propose(args, reply)
}

//Functions related to Testing
func (ps *paxosServer) CheckKill(tester *Tester, currStage string, currTime string) {
	//  sendPropose, sendAccept, sendCommit, receivePropose
	//  receiveAccept, receiveCommit
	//	killTime string //start, mid, end

	if tester.KillStage == currStage && tester.KillTime == currTime {
		fmt.Println("Killing", ps.Port, "Need to stop at", tester.KillStage, tester.KillTime,
			"Stopping at", currStage, currTime)
		ps.listener.Close()
		for _, conn := range ps.RPCConnections {
			conn.Close()
		}
	}
}


//Functions related to Recovery
func (ps *paxosServer) SendRecover() error {
	logs := make(map[int]*RecoverReplyArgs)
	var maxRound int

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
	fmt.Println("in Propose")
	ps.ProposalID++
	proposeReply := &ProposeReplyArgs{}
	majority := ps.NumNodes/2 + ps.NumNodes%2
	var maxPair *KeyValuePair

	for _, conn := range ps.RPCConnections {
		proposeArgs := &ProposeArgs{
			RoundID:    ps.RoundID,
			ProposalID: ps.ProposalID,
			Proposer:   ps.Port,
		}

		err := conn.Call("PaxosServer.HandleProposeRequest", proposeArgs, proposeReply)
		if err != nil {
			fmt.Println("Error while calling HandleProposeRequest", err)
			return err
		}
		fmt.Println("Proposing reply port", proposeReply.AcceptorPort, "round ", proposeReply.RoundID, "accepted? ",
			proposeReply.Accepted, "soumya is ", proposeReply.Pair)

		if time.Now().UnixNano()%2 == 0 {
			ps.CheckKill(&args.Tester, "sendPropose", "mid")
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
		if maxPair != nil {
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
			ps.AcceptedQueue.PushBack(acceptReply)
		}
	}

	if ps.AcceptedQueue.Len() >= majority {
		ps.CheckKill(tester, "sendAccept", "end")
		return ps.SendCommit(acceptors, value, tester)
	}
	return errors.New("Couldn't reach a majority to send the accept requests")
}

func (ps *paxosServer) SendCommit(acceptors *list.List, value []byte, tester *Tester) error {
	ps.CheckKill(tester, "sendCommit", "start")
	fmt.Println("Send Commit")
	for e := acceptors.Front(); e != nil; e = e.Next() {
		reply := e.Value.(*ProposeReplyArgs)
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
	ps.CheckKill(tester, "sendCommit", "end")

	return nil
}

//Functions related to Acceptor
func (ps *paxosServer) HandleProposeRequest(args *ProposeArgs, reply *ProposeReplyArgs) error {
	fmt.Println("Handle Propose Request")
	reply.RoundID = ps.RoundID
	reply.AcceptorPort = ps.Port

	if args.RoundID < ps.RoundID {
		reply.Accepted = false
		return nil
	} else if args.RoundID > ps.RoundID {
		err := ps.SendRecover()
		if err != nil {
			fmt.Println("Paxos Server", ps.Port, "was behind and couldn't recover properly")
		}
	} else if ps.MaxPromisedID >= args.ProposalID {
		reply.Accepted = false
		return nil
	}
	reply.Accepted = true
	if ps.ToCommitQueue.Len() > 0 && ps.ToCommitQueue.Front() != nil {
		commitMsg := ps.ToCommitQueue.Front().Value.(KeyValuePair)
		fmt.Println("queue length and other things : ", ps.ToCommitQueue.Len(), commitMsg.ProposalID, commitMsg.Value)
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
	fmt.Println("handle commit message")

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

