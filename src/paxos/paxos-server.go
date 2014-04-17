package paxos

import (
	"errors"
	"net"
	"strconv"
	"time"
	"sync"
	"rpc"
	"http"
	"sync"
	"net/rpc"
	"net/http"
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

	paxosConnections map[int] *rpc.Client
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
 * Creates a new paxos server and starts the paxos server. masterServerHostPort is the
 * master paxos server which is used to ensure all the paxos servers join the ring. If
 * masterHostPort is empty then it is the master; otherwise, this server is a regular
 * paxos server. port is the the port number that this server should listen on.
 *
 * This function should return only once all paxos servers have joined the ring
 * and should return a non-nil error if the storage server could not be started
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

	//dialing to all other paxos servers and storing the connection
	for i:=0; i<numNodes; i++ {
		currPort := paxosServer.serverRing.servers[i]
		if currPort == port {
			continue
		} else {
			serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+currPort)
			if dialErr != nil {
				return paxosServer, dialErr
			} else {
				paxosServer.paxosConnections[currPort] = serverConn
			}
		}
	}

	return paxosServer, err
}


/* RegisterServer adds a paxos server to the ring. It replies with an error if
 * not all the paxos servers have joined. Once all servers have joined it
 * returns a nil error and a list of all the ports the paxos servers are on.
 */
func (ps *PaxosServer) RegisterServer(args *RegisterArgs, reply *RegisterReply){

	ps.serverRingLock.Lock()
	for i:= 0; i< len(ps.serverRing.servers); i++ {
		if(ps.serverRing.servers[i] == args.port) {
			ps.serverRingLock.Unlock()
			reply.err = errors.New("ALready Registered this server")
			return
		}
	}

	reply.err = nil
	reply.servers = ps.serverRing.servers
	ps.serverRing.servers[ps.serverRing.numConnected] = args.port
	ps.serverRingLock.Unlock()
}


/* GetServers retrieves a list of all connected paxos servers in the ring.
 * It replies with ready equal to false if not all nodes have joined
 * and otherwise with with ready equal to true and the server list
 */
func (ps *PaxosServer) GetServers(args GetServersArgs, reply GetServersReply){
	if ps.serverRing.numConnected == ps.numNodes {
		reply.servers = ps.serverRing.servers
		reply.ready = true
	} else {
		reply.ready = false
	}
}

/* Propose Request should make a rpc call to all the other paxos servers
 * with a proposal ID.
 */
func (ps *PaxosServer) ProposeRequest() error{
	if ps.recentPropNum > ps.lastPropNum {
		ps.recentPropNum++
	} else {
		ps.recentPropNum = ps.lastPropNum + 1
	}

	requestArgs := ProposeRequestArgs{ps.recentPropNum}

	for _, conn := range ps.paxosConnections {
		conn.Call("PaxosServer.HandleProposeRequest", requestArgs)
	}

	return nil
}

/* 
 *
 */
func (ps *PaxosServer) HandleProposeRequest() error{
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleProposeResponse() error {
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleAcceptRequest() error{
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleAcceptResponse() error{
	//TODO
	return errors.New("not implemented")
}

/*
 *
 */
func (ps *PaxosServer) HandleCommit() error{
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

	errRegister := rpc.RegisterName("PaxosServer", ps)
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
		args := &RegisterArgs{"localhost:" + strconv.Itoa(ps.port)}
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
