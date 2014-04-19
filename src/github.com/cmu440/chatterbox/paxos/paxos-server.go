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
	proposalID int
	port int
}

type ProposeResponseArgs struct {
	acceptPropose bool
	previousProposalId int
	previousValue []byte
	port int
}

type AcceptRequestArgs struct {
	proposalId int
	value []byte
	port int
}

type AcceptResponseArgs struct {
	accepted bool
	port int
}

type CommitArgs struct {
	value []byte
	port int
}

type KeyValue struct {
	proposalID int
	value []byte
}

type DefaultReply struct {

}

type PaxosServer struct {
	//TODO
	numNodes int

	receivedMessages *list.List
	proposedMessage KeyValue
	toCommit *list.List


	highestID int
	lastProposedID int

	port int
	masterHostPort string

	serverRing PaxosRing
	serverRingLock *sync.Mutex

	proposeAgainWaitTime int

	paxosConnections map[int] *rpc.Client

	receivedProposeResponses map[*KeyValue] *list.List
	numProposeResponsesReceived int

	receivedAcceptResponses *list.List
	numAcceptResponsesReceived int

	logger *logger.Logger

}

type PaxosRing struct{
	servers []int
	masterHostPort string
	numConnected int
}

type RegisterArgs struct {
	port int
}

type RegisterReply struct {
	err error
	servers []int
}

type GetServersArgs struct {
	//intentionally left empty
}

type GetServersReply struct {
	Ready bool
	Servers []int
}

type SendMessageArgs struct {
	message []byte
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
		servers : make([]int, numNodes),
		masterHostPort : masterHostPort,
		numConnected : 0,
	}

	paxosServer := PaxosServer{
		numNodes : numNodes,
		port : port,
		masterHostPort : masterHostPort,

		receivedMessages :  list.New(), //add messages to this list as they come from chat client
		proposedMessage :  KeyValue{}, //once this proposer becomes a leader and it sends out an accept request
							   //and the key value pair to this list of what to accept

		highestID : 0,		   //highest id seen so far
		lastProposedID : 0,	   //id of the last proposal

		serverRing : serverRing,
		serverRingLock : &sync.Mutex{},

		proposeAgainWaitTime : 0,

		paxosConnections : make(map[int] *rpc.Client),

		receivedProposeResponses : make(map[*KeyValue] *list.List),
		numProposeResponsesReceived : 0,

		receivedAcceptResponses : list.New(),
		numAcceptResponsesReceived : 0,

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
		currPort := paxosServer.serverRing.servers[i]
		fmt.Println("HERE", currPort)

		if currPort == port {
			fmt.Println("already connected so ignore")
			continue //already connected
		} else {
			serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+ strconv.Itoa(currPort))
			if dialErr != nil {
				return &paxosServer, dialErr
			} else {
				paxosServer.paxosConnections[currPort] = serverConn
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
	ps.serverRingLock.Lock()
	for i:= 0; i< ps.serverRing.numConnected; i++ {
		if(ps.serverRing.servers[i] == args.port) {
			alreadyJoined = true
		}
	}

	var err error
	reply.err = nil
	reply.servers = ps.serverRing.servers
	fmt.Println(ps.serverRing.numConnected, args.port)

	if !alreadyJoined {
		ps.serverRing.servers[ps.serverRing.numConnected] = args.port
		ps.serverRing.numConnected++
	}

	ps.serverRingLock.Unlock()

	if ps.serverRing.numConnected != ps.numNodes {
		err = errors.New("Not all servers have joined")
		reply.err = err
	}

	return err
}


/* GetServers retrieves a list of all connected paxos servers in the ring.
 * It replies with ready equal to false if not all nodes have joined
 * and otherwise with with ready equal to true and the server list
 */
func (ps *PaxosServer) GetServers(_ *GetServersArgs, reply *GetServersReply) error{

	if ps.serverRing.numConnected == ps.numNodes {
		reply.Servers = ps.serverRing.servers
		reply.Ready = true
	} else {
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
	value := args.message

	if ps.lastProposedID > ps.highestID {
		ps.lastProposedID++
	} else {
		ps.lastProposedID = ps.highestID+ 1
	}

	requestArgs := ProposeRequestArgs{ps.lastProposedID, ps.port}

	ps.proposedMessage = KeyValue{ps.lastProposedID, value}

	for _, conn := range ps.paxosConnections {
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
	if args.proposalID < ps.lastProposedID {
		reply.acceptPropose = false
		return nil
	} else {
		reply.acceptPropose = true

		if ps.toCommit.Len() == 0 {
			reply.previousProposalId = -1
			reply.previousValue = nil
			return nil
		} else {
			reply.previousProposalId = ps.toCommit.Front().Value.(KeyValue).proposalID
			reply.previousValue = ps.toCommit.Front().Value.(KeyValue).value
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

	majority := (ps.numNodes/2 + ps.numNodes %2) - 1

	ps.numProposeResponsesReceived++

	pair := &KeyValue{args.previousProposalId, args.previousValue}

	if _, ok := ps.receivedProposeResponses[pair]; !ok {
		ps.receivedProposeResponses[pair] = list.New()
	}

	ps.receivedProposeResponses[pair].PushBack(args)


	acceptRequest := AcceptRequestArgs{pair.proposalID, pair.value, args.port}

	if ps.receivedProposeResponses[pair].Len() >= majority {
		//go through all the received propose responses and send them an accept request

		for e:= ps.receivedProposeResponses[pair].Front(); e!=nil; e = e.Next(){

			port := e.Value.(ProposeResponseArgs).port

			if port == ps.port {
				continue
			}

			conn := ps.paxosConnections[port]

			reply := &AcceptResponseArgs{}
			err := conn.Call("PaxosServer.HandleAcceptRequest", &acceptRequest, reply)

			if (err != nil) {
				return err
			}else {
				return nil
			}
		}

		ps.toCommit.PushBack(acceptRequest)
		return nil
	}

	//if we've received responses from everybody and have not reached majority
	//retry propose

	if ps.numProposeResponsesReceived == ps.numNodes -1 {

		val := ps.toCommit.Front().Value.(KeyValue).value
		ps.toCommit.Remove(ps.toCommit.Front())

		err := ps.ProposeRequest(&SendMessageArgs{val}, &DefaultReply{})

		for err != nil {
			time.Sleep(time.Duration(ps.proposeAgainWaitTime) * time.Millisecond)
			ps.proposeAgainWaitTime = ps.proposeAgainWaitTime * 2
			err = ps.ProposeRequest(&SendMessageArgs{val}, &DefaultReply{})
		}

		ps.proposeAgainWaitTime = 1
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

	if ps.highestID > args.proposalId {
		reply.accepted = false
	} else {
		reply.accepted = true
		reply.port = ps.port
	}

	//send the reply back to the proposer with rpc call
	ps.toCommit.PushBack(args)

	err := ps.paxosConnections[args.port].Call("PaxosServer.HandleAcceptResponse", &reply, &DefaultReply{})

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
	majority := (ps.numNodes/2 + ps.numNodes %2) - 1

	ps.numAcceptResponsesReceived ++

	if args.accepted {
		ps.receivedAcceptResponses.PushBack(args)
	}

	if ps.receivedAcceptResponses.Len() >= majority {
		//TODO send commit message to everyone in the majority :)
		//go through everyone in the majority, get each port (ps.port) and value (front of the toCommit list)
		//and then send it to them in an rpc call

		commitMsg := &CommitArgs{ps.toCommit.Front().Value.(AcceptRequestArgs).value, ps.port}

		for e:=ps.receivedAcceptResponses.Front(); e!=nil; e=e.Next(){
			port := e.Value.(AcceptResponseArgs).port
			conn := ps.paxosConnections[port]

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
	//
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
		return errRegister
	}

	fmt.Println("the port in master is ", ps.port)
	listener, errListen := net.Listen("tcp", ":"+strconv.Itoa(ps.port))
	if errListen != nil {
		return errListen
	}

	go http.Serve(listener, nil)


	numTries := 0

	for {
		fmt.Println("Master Trying", numTries)
		numTries++

		args := &RegisterArgs{ps.port}

		servers := make([]int, ps.numNodes)

		reply := &RegisterReply{nil, servers}

		ps.RegisterServer(args, reply)

		if reply.err != nil {
			fmt.Println(reply.err)
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
	errRegister := rpc.RegisterName("PaxosServer", ps)
	rpc.HandleHTTP()
	fmt.Println("handling http")

	if errRegister != nil {
		return errRegister
	}

	fmt.Println("the port is ", ps.port)
	listener, errListen := net.Listen("tcp", ":"+strconv.Itoa(ps.port))

	if errListen != nil {
		return errListen
	}

	go http.Serve(listener, nil)

	fmt.Println("Masterhost port", ps.masterHostPort)
	slave, errDial := rpc.DialHTTP("tcp", ps.masterHostPort)

	if errDial != nil {
		return errDial
	}

	numTries := 0

	for {
		numTries++
		args := &RegisterArgs{ps.port}
		reply := &RegisterReply{}

		fmt.Println("TRYINGGG", args.port)
		errCall := slave.Call("PaxosServer.RegisterServer", args, reply)
		fmt.Println("HEREEE")

		if errCall != nil {
			return errCall
		}

		if reply.err != nil {
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
		ps.serverRing.servers = reply.Servers
		return nil
	}

	return errors.New("Was unable to get servers even after all paxos servers joined the ring")
}
