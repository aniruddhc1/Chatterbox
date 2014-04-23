package multipaxos

/* TODO
	things left to do for paxos implentation
	1. if the proposer is still leader after committing propose the next message in the MsgQueue
	2. Write the function SendMessage (basically a wrapper around paxos) which gets the message
	   from the chat client and receives an error if any step of the paxos process fails.
	   If it fails make it start the paxos again
	3. do the learner functions at the bottom.
 */


import (
	"container/list"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
	"bytes"
)

type KeyValuePair struct {
	ProposalID   int
	Value []byte
}

type GetServersArgs struct {
	//intentionally left empty
}

type GetServersReply struct {
	Ready bool
	Servers []int
}

type SendMessageArgs struct {
	Value []byte
}

type SendMessageReplyArgs struct {
}


type ProposeArgs struct {
	RoundID    int
	ProposalID int
	Proposer   int //port of the proposer
}

type ProposeReplyArgs struct {
	RoundID      int
	Pair *KeyValuePair //consists of proposal id and value
	Accepted     bool
	AcceptorPort int
}

type AcceptRequestArgs struct {
	RoundID    int
	ProposalID int
	Value      []byte
}

type AcceptReplyArgs struct {
	Accepted bool
	AcceptorPort int
	RoundID	 int
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
	value   []byte
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

	//Required for proposer role
	ProposalID           int
	MsgQueue             *list.List
	ProposeAcceptedQueue *list.List //contains the ProposeReplyArgs received from all the acceptors who have accepted the proposal; Reset at each round
	AcceptedQueue        *list.List //contains the AcceptReplyArgs received rom all acceptors who have accepted the Accept request; reset after every round

	//Required for accepter role
	ToCommitQueue *list.List //list of proposalID-data keyvaluepair of accepted but yet not committed messages
	MaxPromisedID int

	//Required for learner role
	CommittedMsgs map[int][]byte //message committed for every round of Paxos
}

func NewPaxosServer(masterHostPort string, numNodes, port int) (*paxosServer, error) {

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
		CommittedMsgs: make(map[int][]byte),
	}

	//Register the server to RPC
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

//TODO functions requred for chat client including GetServers and SendMessage

func (ps *paxosServer) SendMessage(args *SendMessageArgs, reply *SendMessageReplyArgs) error {
	ps.MsgQueue.PushBack(args.Value)
	return ps.Propose(args, reply)
}

//Functions related to Proposer
func (ps *paxosServer) Propose(args *SendMessageArgs, _ *SendMessageReplyArgs) error  {
	fmt.Println("Propose")
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
		if proposeReply.RoundID != ps.RoundID {
			//TODO Do recover
		} else if proposeReply.Accepted {
			ps.ProposeAcceptedQueue.PushBack(proposeReply)
			if proposeReply.Pair != nil && proposeReply.Pair.ProposalID > maxPair.ProposalID {
				maxPair.ProposalID = proposeReply.Pair.ProposalID
				maxPair.Value = proposeReply.Pair.Value
			}
		} else {
			continue
		}
	}

	if ps.ProposeAcceptedQueue.Len() >= majority {
		if maxPair != nil {
			return ps.SendAcceptRequests(ps.ProposeAcceptedQueue, maxPair.ProposalID, maxPair.Value)
		} else {
			return ps.SendAcceptRequests(ps.ProposeAcceptedQueue, ps.ProposalID, args.Value)
		}
	}

	return errors.New("Couldn't get a majority to accept request to proposal")
}

func (ps *paxosServer) SendAcceptRequests(acceptors *list.List, id int, value []byte) error {
	fmt.Println("Sending Accept Requests")
	majority := ps.NumNodes/2 + ps.NumNodes%2

	for e := acceptors.Front(); e != nil; e = e.Next() {
		reply := e.Value.(ProposeReplyArgs)
		conn := ps.RPCConnections[reply.AcceptorPort]

		acceptArgs := &AcceptRequestArgs{
			RoundID:    ps.RoundID,
			ProposalID: id,
			Value:      value,
		}
		acceptReply := &AcceptReplyArgs{}

		err := conn.Call("PaxosServer.HandleAcceptRequest", acceptArgs, acceptReply)
		if err != nil {
			fmt.Println("error in send accept request ", err)
		}

		if acceptReply.Accepted {
			ps.AcceptedQueue.PushBack(acceptReply)
		}
	}

	if ps.AcceptedQueue.Len() >= majority {
		return ps.SendCommit(acceptors, value)
	}

	return errors.New("Couldn't reach a majority to send the accept requests")
}

func (ps *paxosServer) SendCommit(acceptors *list.List, value []byte) error {
	fmt.Println("Send Commit")
	for e := acceptors.Front(); e != nil; e = e.Next() {
		reply := e.Value.(ProposeReplyArgs)
		conn := ps.RPCConnections[reply.AcceptorPort]

		commitArgs := &CommitArgs{Value : value, RoundID : ps.RoundID}
		commitReply := &CommitReplyArgs{}

		err := conn.Call("PaxosServer.HandleCommit", commitArgs, commitReply)

		if err != nil {
			fmt.Println("error in send commit request ", err)
		}
	}

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
		//TODO recover state
	} else if ps.MaxPromisedID >= args.ProposalID {
		reply.Accepted = false
		return nil
	}
	reply.Accepted = true
	commitMsg := ps.ToCommitQueue.Front().Value.(KeyValuePair)
	reply.Pair = &KeyValuePair{commitMsg.ProposalID, commitMsg.Value}
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
		//TODO recover state
	} else if ps.MaxPromisedID >= args.ProposalID {
		reply.Accepted = false
		return nil
	}

	ps.ToCommitQueue.PushBack(KeyValuePair{args.ProposalID, args.Value})
	reply.Accepted = true
	return nil
}

func (ps *paxosServer) HandleCommit(args *CommitArgs, reply *CommitReplyArgs) error {

	fmt.Println("sending commit message")

	ps.CommittedMsgs[args.RoundID] = args.Value

	if ps.RoundID < args.RoundID {
		//TODO recover state
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

func (ps *paxosServer) GetServers(_ *GetServersArgs, reply *GetServersReply) error{

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

//Functions related to Learner

/*TODO


 */
