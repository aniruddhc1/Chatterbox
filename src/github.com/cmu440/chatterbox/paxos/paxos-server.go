package paxos

import (
	"errors"
	"net"
	"strconv"
	"time"
	"sync"
	"net/rpc"
	"net/http"
	"container/list"
	"github.com/cmu440/chatterbox/logger"
	"fmt"
)


type ProposeRequestArgs struct {
	ProposalID int
	Port int
}

type ProposeResponseArgs struct {
	AcceptPropose bool
	PreviousProposalId int
	PreviousValue []byte
	Port int
}

type AcceptRequestArgs struct {
	ProposalID int
	Value []byte
	Port int
}

type AcceptResponseArgs struct {
	Accepted bool
	Port int
}

type CommitArgs struct {
	Value []byte
	Port int
}

type KeyValue struct {
	ProposalID int
	Value []byte
}

type DefaultReply struct {

}

type PaxosServer struct {
	//TODO
	NumNodes int

	ReceivedMessages *list.List
	ProposedMessage KeyValue
	ToCommit *list.List


	HighestID int
	LastProposedID int

	Port int
	MasterHostPort string

	ServerRing PaxosRing
	ServerRingLock *sync.Mutex

	ProposeAgainWaitTime int

	PaxosConnections map[int] *rpc.Client

	ReceivedProposeResponses map[*KeyValue] *list.List
	NumProposeResponsesReceived int

	ReceivedAcceptResponses *list.List
	NumAcceptResponsesReceived int

	Logger *logger.Logger

}

type PaxosRing struct{
	Servers []int
	MasterHostPort string
	NumConnected int
}

type RegisterArgs struct {
	Port int
}

type RegisterReply struct {
	Err error
	Servers []int
}

type GetServersArgs struct {
	//intentionally left empty
}

type GetServersReply struct {
	Ready bool
	Servers []int
}

type SendMessageArgs struct {
	Message []byte
}

/*
 *
 * Creates a new paxos server and starts the paxos server. masterServerHostPort is the
 * master paxos server which is used to ensure all the paxos servers join the ring.
 * masterHostPort is the master server's hostport. If masterHostPort is empty then
 * it is the master; otherwise, this server is a regular paxos server. port is
 * the port number that this server should listen on.
 * This function should return only once all paxos servers have joined the ring
 * and should return a non-nil error if the paxos server could not be started
 *
 */
func NewPaxosServer(masterHostPort string, numNodes, port int) (*PaxosServer, error) {
	//TODO

	serverRing := PaxosRing{
		Servers : make([]int, numNodes),
		MasterHostPort : masterHostPort,
		NumConnected : 0,
	}

	paxosServer := PaxosServer{
		NumNodes : numNodes,
		Port : port,
		MasterHostPort : masterHostPort,

		ReceivedMessages :  list.New(), //add messages to this list as they come from chat client
		ProposedMessage :  KeyValue{}, //once this proposer becomes a leader and it sends out an accept request
							   //and the key value pair to this list of what to accept

		HighestID : 0,		   //highest id seen so far
		LastProposedID : 0,	   //id of the last proposal

		ServerRing : serverRing,
		ServerRingLock : &sync.Mutex{},

		ProposeAgainWaitTime : 0,

		PaxosConnections : make(map[int] *rpc.Client),

		ReceivedProposeResponses : make(map[*KeyValue] *list.List),
		NumProposeResponsesReceived : 0,

		ReceivedAcceptResponses : list.New(),
		NumAcceptResponsesReceived : 0,

		//logger : logger.NewLogger(), //TODO
	}

	var err error

	if masterHostPort == "" {
		err = paxosServer.startMaster()
	} else {
		err = paxosServer.startServer()
	}

	fmt.Println("GOT HERE")
	//dialing to all other paxos servers and storing the connection

	for i:=0; i<numNodes; i++ {
		currPort := paxosServer.ServerRing.Servers[i]
		fmt.Println("HERE", currPort)

		if currPort == port {
			fmt.Println("already connected so ignore")
			continue //already connected
		} else {
			serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+ strconv.Itoa(currPort))
			if dialErr != nil {
				return &paxosServer, dialErr
			} else {
				paxosServer.PaxosConnections[currPort] = serverConn
			}
		}
	}
	return &paxosServer, err
}


/* RegisterServer adds a paxos server to the ring. It replies with an error if
 * not all the paxos servers have joined. Once all servers have joined it
 * returns a nil error and a list of all the ports the paxos servers are on.
 */
func (ps *PaxosServer) RegisterServer(args *RegisterArgs, reply *RegisterReply) error{
	fmt.Println("REGISTERING")
	alreadyJoined := false
	ps.ServerRingLock.Lock()
	for i:= 0; i< ps.ServerRing.NumConnected; i++ {
		if(ps.ServerRing.Servers[i] == args.Port) {
			alreadyJoined = true
		}
	}

	var err error
	reply.Err = nil
	reply.Servers = ps.ServerRing.Servers
	fmt.Println(ps.ServerRing.NumConnected, args.Port)

	if !alreadyJoined {
		fmt.Println("IN REGISTER", ps.ServerRing.NumConnected, args.Port)
		ps.ServerRing.Servers[ps.ServerRing.NumConnected] = args.Port
		ps.ServerRing.NumConnected++
	}

	ps.ServerRingLock.Unlock()

	if ps.ServerRing.NumConnected != ps.NumNodes {
		err = errors.New("Not all servers have joined")
		reply.Err = err
		return err
	}

	return err
}


/* GetServers retrieves a list of all connected paxos servers in the ring.
 * It replies with ready equal to false if not all nodes have joined
 * and otherwise with with ready equal to true and the server list
 */
func (ps *PaxosServer) GetServers(_ *GetServersArgs, reply *GetServersReply) error{

	if ps.ServerRing.NumConnected == ps.NumNodes {
		reply.Servers = ps.ServerRing.Servers
		fmt.Println("number of nodes connected are ", ps.ServerRing.NumConnected)
		reply.Ready = true
	} else {
		fmt.Println("number of nodes connected are ", ps.ServerRing.NumConnected)
		reply.Ready = false
	}

	return nil
}

/*
 * This is what the chat client will call
 */
func (ps *PaxosServer) SendMessage(args *SendMessageArgs, reply *DefaultReply) error {
	//TODO
    return nil
}

/* Propose Request should make a rpc call to all the other paxos servers
 * with a proposal ID.
 */
func (ps *PaxosServer) ProposeRequest(args *SendMessageArgs, reply *DefaultReply) error{
	value := args.Message

	if ps.LastProposedID > ps.HighestID {
		ps.LastProposedID++
	} else {
		ps.LastProposedID = ps.HighestID+ 1
	}

	requestArgs := ProposeRequestArgs{ps.LastProposedID, ps.Port}

	ps.ProposedMessage = KeyValue{ps.LastProposedID, value}

	for _, conn := range ps.PaxosConnections {
		conn.Call("PaxosServer.HandleProposeRequest", requestArgs, &DefaultReply{})
	}

	return nil
}

/*
 * if the acceptor has already seen a proposal with a higher proposal id
 * reject the proposal else, send back the previously seen message to commit
 */
func (ps *PaxosServer) HandleProposeRequest(args *ProposeRequestArgs, _ *DefaultReply) error{
	//TODO
	reply := ProposeResponseArgs{}

	//check if the proposal id larger than the last seen proposal id
	if args.ProposalID < ps.LastProposedID {
		reply.AcceptPropose = false
		return nil
	} else {
		reply.AcceptPropose = true

		if ps.ToCommit.Len() == 0 {
			reply.PreviousProposalId = -1
			reply.PreviousValue = nil
			return nil
		} else {
			reply.PreviousProposalId = ps.ToCommit.Front().Value.(KeyValue).ProposalID
			reply.PreviousValue = ps.ToCommit.Front().Value.(KeyValue).Value
		}
	}

	//TODO
	// make a rpc call back to the server in the args port

	return errors.New("unkown")
}

/*
 * Need to wait for majority so if the response creates a majority then send out
 * accept requests to all acceptors and update the toCommit list
 */
func (ps *PaxosServer) HandleProposeResponse(args *ProposeResponseArgs, reply *DefaultReply) error {
	//TODO

	//TODO might have to do some weird stuff with the nil responses because we
	//cant have a nil key in the map

	majority := (ps.NumNodes/2 + ps.NumNodes %2) - 1

	ps.NumProposeResponsesReceived++

	pair := &KeyValue{args.PreviousProposalId, args.PreviousValue}

	if _, ok := ps.ReceivedProposeResponses[pair]; !ok {
		ps.ReceivedProposeResponses[pair] = list.New()
	}

	ps.ReceivedProposeResponses[pair].PushBack(args)


	acceptRequest := AcceptRequestArgs{pair.ProposalID, pair.Value, args.Port}

	if ps.ReceivedProposeResponses[pair].Len() >= majority {
		//go through all the received propose responses and send them an accept request

		for e:= ps.ReceivedProposeResponses[pair].Front(); e!=nil; e = e.Next(){

			port := e.Value.(ProposeResponseArgs).Port

			if port == ps.Port {
				continue
			}

			conn := ps.PaxosConnections[port]

			reply := &AcceptResponseArgs{}
			err := conn.Call("PaxosServer.HandleAcceptRequest", &acceptRequest, reply)

			if (err != nil) {
				return err
			}else {
				return nil
			}
		}

		ps.ToCommit.PushBack(acceptRequest)
		return nil
	}

	//if we've received responses from everybody and have not reached majority
	//retry propose

	if ps.NumProposeResponsesReceived == ps.NumNodes -1 {

		val := ps.ToCommit.Front().Value.(KeyValue).Value
		ps.ToCommit.Remove(ps.ToCommit.Front())

		err := ps.ProposeRequest(&SendMessageArgs{val}, &DefaultReply{})

		for err != nil {
			time.Sleep(time.Duration(ps.ProposeAgainWaitTime) * time.Millisecond)
			ps.ProposeAgainWaitTime = ps.ProposeAgainWaitTime * 2
			err = ps.ProposeRequest(&SendMessageArgs{val}, &DefaultReply{})
		}

		ps.ProposeAgainWaitTime = 1
	}
	return errors.New("not implemented")
}

/* reply with true if a higher value has not been seen and otherwise
 * reply with false
 */
func (ps *PaxosServer) HandleAcceptRequest(args *AcceptRequestArgs, _ *DefaultReply) error{
	//check that you haven't received a higher proposal id than the one given
	//if it is send an error back not accepting
	reply := &AcceptResponseArgs{}

	if ps.HighestID > args.ProposalID {
		reply.Accepted = false
	} else {
		reply.Accepted = true
		reply.Port = ps.Port
	}

	//send the reply back to the proposer with rpc call
	ps.ToCommit.PushBack(args)

	err := ps.PaxosConnections[args.Port].Call("PaxosServer.HandleAcceptResponse", &reply, &DefaultReply{})

	if err != nil{
		return err
	}

	return nil

}

/* wait for majority. Once majority has been reached then send a commit message
 * if all proposers have replied and majority hasnt been reached then try proposing
 * again. Otherwise send a commit message with key and value to all nodes, and send
 * back the value to the chat client
 */
func (ps *PaxosServer) HandleAcceptResponse(args *AcceptResponseArgs, reply *DefaultReply) error{
	//TODO
	majority := (ps.NumNodes/2 + ps.NumNodes %2) - 1

	ps.NumAcceptResponsesReceived ++

	if args.Accepted {
		ps.ReceivedAcceptResponses.PushBack(args)
	}

	if ps.ReceivedAcceptResponses.Len() >= majority {
		//TODO send commit message to everyone in the majority :)
		//go through everyone in the majority, get each port (ps.port) and value (front of the toCommit list)
		//and then send it to them in an rpc call

		commitMsg := &CommitArgs{ps.ToCommit.Front().Value.(AcceptRequestArgs).Value, ps.Port}

		for e:=ps.ReceivedAcceptResponses.Front(); e!=nil; e=e.Next(){
			port := e.Value.(AcceptResponseArgs).Port
			conn := ps.PaxosConnections[port]

			err := conn.Call("PaxosServer.HandleCommit", &commitMsg, &DefaultReply{}) //todo agree that reply should be nothing
			if err != nil {
				return err
			}
		}
	}

	//TODO after chat client code
	//let chat client know that message has been committed.
	return errors.New("not implemented")
}


/*
 *
 */
func (ps *PaxosServer) HandleCommit(args *CommitArgs, reply *DefaultReply) error{
	args = args
	//TODO
	//Basically for now just log, idk what else to do here

	//Reset everything
//	for ps.toCommit.



	return errors.New("not implemented")
}





//													//
//				HELPER FUNCTIONS BELOW				//
//													//


/*
 *
 */
func (ps *PaxosServer) startMaster() error {

	errRegister := rpc.RegisterName("PaxosServer", ps)
	rpc.HandleHTTP()

	if errRegister != nil {
		fmt.Println("An error occured while doing rpc register", errRegister)
		return errRegister
	}

	fmt.Println("the port in master is ", ps.Port)
	listener, errListen := net.Listen("tcp", ":"+strconv.Itoa(ps.Port))
	if errListen != nil {
		fmt.Println(errListen)
		return errListen
	}

	go http.Serve(listener, nil)


	numTries := 0

	for {
		fmt.Println("Master Trying", numTries)
		numTries++

		args := &RegisterArgs{ps.Port}

		servers := make([]int, ps.NumNodes)

		reply := &RegisterReply{nil, servers}

		ps.RegisterServer(args, reply)

		fmt.Println("NumNodes", ps.NumNodes, ps.ServerRing.NumConnected)

		if ps.ServerRing.NumConnected == ps.NumNodes {
			break
		}

		if reply.Err != nil {
			fmt.Println(reply.Err)
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	return nil
}

/*
 *
 */
func (ps *PaxosServer) startServer() error {
	fmt.Println("IN START SLAVE")
	errRegister := rpc.RegisterName("PaxosServer", ps)

	rpc.HandleHTTP()
	fmt.Println("handling http")

	if errRegister != nil {
		return errRegister
	}

	fmt.Println("the port is ", ps.Port)
	listener, errListen := net.Listen("tcp", ":"+strconv.Itoa(ps.Port))

	if errListen != nil {
		fmt.Println("Err occured while listenening", errListen)
		return errListen
	}

	go http.Serve(listener, nil)

	fmt.Println("Masterhost port", ps.MasterHostPort)



	numTries := 0
	dialSuccess := false

	var slave *rpc.Client
	var errDial error

	for {
		if !dialSuccess {
			slave, errDial = rpc.DialHTTP("tcp", ps.MasterHostPort)

			if errDial != nil {
				fmt.Println(errDial)
				time.Sleep(time.Second)
				continue
			}
		}

		dialSuccess = true
		numTries++
		args := &RegisterArgs{ps.Port}
		reply := &RegisterReply{}

		fmt.Println("TRYINGGG", args.Port)
		errCall := slave.Call("PaxosServer.RegisterServer", args, reply)
		fmt.Println("HEREEE")

		fmt.Println("The errcall is", errCall)

		if errCall != nil {
			fmt.Println(errCall)
			return errCall
		}

		if reply.Err != nil {
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	reply := &GetServersReply{}
	err := slave.Call("PaxosServer.GetServers", &GetServersArgs{}, reply)

	if err != nil {
		return err
	}

	if reply.Ready {
		ps.ServerRing.Servers = reply.Servers
		return nil
	}

	return errors.New("Was unable to get servers even after all paxos servers joined the ring")
}
