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
	Key   int
	Value []byte
}

type ProposeArgs struct {
	RoundID    int
	ProposalID int
	Proposer   int //port of the proposer
}

type ProposeReplyArgs struct {
	RoundID      int
	ProposalID   int
	Value        []byte
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

type PaxosServer struct {
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

func NewPaxosServer(masterHostPort string, numNodes, port int) (*PaxosServer, error) {

	//Initialize paxos server
	paxosServer := &PaxosServer{
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
	errRegister := rpc.RegisterName("PaxosServer", paxosServer)
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

func (ps *PaxosServer) RegisterServer(args *RegisterArgs, reply *RegisterReplyArgs) error {
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

//Functions related to Proposer
func (ps *PaxosServer) Propose() error {
	ps.ProposalID++
	proposeReply := &ProposeReplyArgs{}
	majority := ps.NumNodes/2 + ps.NumNodes%2

	for port, conn := range ps.RPCConnections {
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
		} else {
			continue
		}
	}

	if ps.ProposeAcceptedQueue.Len() >= majority {
		//TODO figure which id, and value to send with accept request
		return ps.SendAcceptRequests(ps.ProposeAcceptedQueue)
	}

	return errors.New("Couldn't get a majority to accept request to proposal")
}

func (ps *PaxosServer) SendAcceptRequests(acceptors *list.List, id int, value []byte) error {
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

func (ps *PaxosServer) SendCommit(acceptors *list.List, value []byte) error {
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
func (ps *PaxosServer) HandleProposeRequest(args *ProposeArgs, reply *ProposeReplyArgs) error {
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
	reply.ProposalID = commitMsg.Key
	reply.Value = commitMsg.Value
	return nil
}

func (ps *PaxosServer) HandleAcceptRequest(args *AcceptRequestArgs, reply *AcceptReplyArgs) error {
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

func (ps *PaxosServer) HandleCommit(args *CommitArgs, reply *CommitReplyArgs) error {
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

//Functions related to Learner

/*TODO


 */
