package paxos

import (
	"errors"
	"net/rpc"
	"net"
	"strconv"
	"net/http"
	"time"
	"sync"
)


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
	previousProposalId int
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
	numNodes int

	lastVal []byte
	lastPropNum int
	recentPropNum int

	port string
	masterHostPort string

	serverRing PaxosRing
	serverRingLock *sync.Mutex
}

type PaxosRing struct{
	servers []string
	masterHostPort string
	numConnected int
}

type RegisterArgs struct {
	port string
}

type RegisterReply struct {
	err error
	servers []string
}

type GetServersArgs struct {
	//intentionally left empty
}

type GetServersReply struct {
	ready bool
	servers []string
}


/*
 * creates a new paxos server node.
 * it connects on a port number (port),
 * and stores the current number of nodes (numNodes) so that it knows what the simple majority needed is supposed to be.
 * additionally, we require a masterHostPort
 *
 */
func NewPaxosServer(masterHostPort string, numNodes, port int) (PaxosServer, error) {
	//TODO

	serverRing := PaxosRing{
		servers : make([]string, numNodes),
		masterHostPort : masterHostPort,
		numConnected : 0,
	}

	paxosServer := PaxosServer{
		numNodes : numNodes,
		port : port,
		masterHostPort : masterHostPort,

		lastVal : nil,
		lastPropNum : 0,
		recentPropNum : 0, //max(recentpropnum+1, lastpropnum+1)

		serverRing : serverRing,
		serverRingLock : &sync.Mutex{},
	}

	var err error

	if masterHostPort == "" {
		err = paxosServer.startMaster()
	} else {
		err = paxosServer.startServer()
	}

	return paxosServer, err
}

/*
 *
 */
func (ps *PaxosServer) RegisterServer(args *RegisterArgs, reply *RegisterReply) error {

	ps.serverRingLock.Lock()
	for i:= 0; i< len(ps.serverRing.servers); i++ {
		if(ps.serverRing.servers[i] == args.port) {
			ps.serverRingLock.Unlock()
			return errors.New("ALready Registered this server")
		}
	}

	ps.serverRing.servers[ps.serverRing.numConnected] = args.port
	ps.serverRingLock.Unlock()
	return nil
}

/*
 *
 */
func (ps *PaxosServer) GetServers(args GetServersArgs, reply GetServersReply) error {
	if ps.serverRing.numConnected == ps.numNodes {
		reply.servers = ps.serverRing.servers
		reply.ready = true
	} else {
		reply.ready = false
	}
	return nil
}

/*
 *
 */
func (ps *PaxosServer) ProposeRequest() error{
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleProposeRequest(msg PaxosMessage) error{
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleProposeResponse(msg PaxosMessage) error {
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleAcceptRequest(msg PaxosMessage) error{
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleAcceptResponse(msg PaxosMessage) error{
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleCommit(msg PaxosMessage) error{
	//TODO
	return errors.New("not implemented")
}

/*
 *					HELPER FUNCTIONS BELOW
 */


/*
 *
 */
func (ps *PaxosServer) startMaster() error {

	errRegister := rpc.RegisterName("StorageServer", ps)
	rpc.HandleHTTP()

	if errRegister != nil {
		return errRegister
	}

	listener, errListen := net.Listen("tcp", ":"+strconv.Itoa(ps.port))

	if errListen != nil {
		return errListen
	}

	go http.Serve(listener, nil)

	numTries := 0

	for {
		numTries++

		args := &RegisterArgs{"localhost:" + strconv.Itoa(ps.port)}

		servers := make([] string ps.numNodes)

		reply := &RegisterReply{servers, nil}

		ps.RegisterServer(args, reply)

		if reply.error != nil {
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
	errRegister := rpc.RegisterName("StorageServer", ps)
	rpc.HandleHTTP()

	if errRegister != nil {
		return errRegister
	}
	listener, errListen := net.Listen("tcp", ":"+strconv.Itoa(ps.port))

	if errListen != nil {
		return errListen
	}

	go http.Serve(listener, nil)

	slave, errDial := rpc.DialHTTP("tcp", ps.masterHostPort)

	if errDial != nil {
		return errDial
	}

	numTries := 0

	for {
		numTries++
		args := &RegisterArgs{"localhost:" + strconv.Itoa(ss.port)}
		var reply *storagerpc.RegisterReply

		errCall := slave.Call("StorageServer.RegisterServer", args, &reply)

		if errCall != nil {
			return errCall
		}

		if reply.err != nil {
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	return nil
}
