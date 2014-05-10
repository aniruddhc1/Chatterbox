package multipaxos

/* TODO Things left to do for paxos implentation
1. Write the function SendMessage (basically a wrapper around paxos) which gets the message
   from the chat client and receives an error if any step of the paxos process fails.
   If it fails make it start the paxos again
2. Do the exponential backoff stuff to prevent livelock
3. Change recovery from logs to a complete file based log system
4. Write a lot more tests
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
	"io/ioutil"
	"strings"
)

type paxosServer struct {
	Port           int
	MasterHostPort string
	NumNodes       int
	NumConnected   int                 //number of servers that are currently connected and active
	RPCConnections map[int]*rpc.Client //from port number to the rpc connection so we don't have to dial everytime
	Servers        []int               //port numbers of all the servers in the ring
	listener       *net.Listener
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

	wakeupChan chan bool

}

	var Active bool

func NewPaxosServer(masterHostPort string, numNodes, port int) (*paxosServer, error) {
	fmt.Println("CREATING NEWWWW SERVERRR")
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
		Servers:        make([]int, 0, numNodes),
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

		wakeupChan : make(chan bool),
	}

	Active = true // set to false if need to block all the rpc functions

	//Register the server http://angusmacdonald.me/writing/paxos-by-example/ to RPC
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

	paxosServer.listener = &listener

	go http.Serve(listener, nil)

	//If the server is a slave create a connection to the master
	var regular *rpc.Client
	var errDial error
	if masterHostPort != "" {
		regular, errDial = rpc.DialHTTP("tcp", masterHostPort)
		if errDial != nil {
			//fmt.Println("Slave couldn't connect to master", errDial)
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
			if err != nil {
				//fmt.Println(paxosServer.Port, err, "SLEEPING for SECOND")
				time.Sleep(time.Second)
				continue
			}
		} else {
			//This is a regular paxos server
			err := regular.Call("PaxosServer.RegisterServer", args, reply)
			if err != nil {
				//fmt.Println(paxosServer.Port, err, "SLEEPING for SECOND")
				time.Sleep(time.Second)
				continue
			}
		}

		//fmt.Println("Setting paxos servers", paxosServer.Port, reply.NumConnected)
		paxosServer.Servers = reply.Servers
		break
	}

	paxosServer.NumConnected = paxosServer.NumNodes
	err = paxosServer.CreatePaxosConnections()
	if err!= nil {
		fmt.Println(err)
		return nil, err
	}

	return paxosServer, nil
}

func (ps *paxosServer) CreatePaxosConnections() error{
	//Create rpc connections to all servers
	for i := 0; i < ps.NumNodes; i++ {
		currPort := ps.Servers[i]
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
		fmt.Println("IN REGISTER SERVER", ps.NumConnected, args.Port)

		for i := 0; i < ps.NumConnected; i++ {
			if ps.Servers[i] == args.Port {
				alreadyJoined = true
			}
		}

		if !alreadyJoined {
			fmt.Println("IN REGISTER", ps.NumConnected, args.Port)
			ps.Servers = append(ps.Servers, args.Port)
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

func (ps *paxosServer) ServeMessageFile(args *FileArgs, reply *FileReply) error{
		fmt.Println("given the file")
		fmt.Println(ps.CommittedMsgsFile.Name())
		replyBytes, err := ioutil.ReadFile(ps.CommittedMsgsFile.Name())

		if(err != nil){
			return err
		}
		replyBytes1 := bytes.Trim(replyBytes, "\x00")
		reply.File = replyBytes1

		if err != nil{
			fmt.Println(err)
			return err
		}

	return nil
}

func (ps *paxosServer) SendMessage(args *SendMessageArgs, reply *SendMessageReplyArgs) error {
	if(Active) {
		fmt.Println("in send message")
		ps.MsgQueue.PushBack(args.Value)
		return ps.Propose(args, reply)
	}
	return nil
}

func (ps *paxosServer) WakeupServer(args *WakeupRequestArgs, reply *WakeupReplyArgs) error{
	fmt.Println("Wakingup server", ps.Port)
	ps.wakeupChan <- true
	return nil
}

//Functions related to Testing
func (ps *paxosServer) CheckKill(tester *Tester, currStage string, currTime string) (bool, error) {
	//  sendPropose, sendAccept, sendCommit, receivePropose
	//  receiveAccept, receiveCommit
	//	killTime string //start, mid, end
	fmt.Println("In Check Kill", ps.Port, "stage", currStage, "time", currTime)
	if tester.Stage == currStage && tester.Time == currTime {
		fmt.Println("got into if statement in check kill")
		var timeAmount int
		Active = false

		if tester.Kill {
			timeAmount = 6000 //just a really large number to simulate sleeping
		} else {
			timeAmount = tester.SleepTime
		}

		go func() {
			t := time.NewTimer(time.Duration(timeAmount)* time.Second)
			select {
			case <-t.C:
				Active = true
				fmt.Println("WAKING UP")
				//do nothing
			case <-ps.wakeupChan:
				Active = true
				t.Stop()
			}
		}()

		return true, nil
	}
	return false, nil
}


//Functions related to Recovery
func (ps *paxosServer) SendRecover(args *ManualRecoverArgs, reply *ManualRecoverReply) error {
	if(Active) {
		fmt.Println("bugaboo in handle recover", ps.Port)
		logs := make(map[int]*RecoverReplyArgs)
		maxRound := 0

		for port, conn := range ps.RPCConnections {
			args := &RecoverArgs{ps.RoundID}
			reply := &RecoverReplyArgs{}

			err := conn.Call("PaxosServer.HandleRecover", args, reply)

			if err != nil {
				fmt.Println("Error occured while calling Handle Recover", err)
				continue
			}

			logs[port] = reply
			if maxRound < reply.RoundID {
				maxRound = reply.RoundID
			}
		}

		//check that all logs are consistent and update this servers logs
		for i := ps.RoundID; i < maxRound; i++ {
			//check to make sure that all of them are the same for that round id
			var val []byte
			fmt.Println("RECOVERING ROUND", i)
			for port, log := range logs {
				if log.RoundID > ps.RoundID {
					if val == nil {
						fmt.Println("Got the first log from", port)
						val = log.CommittedValues[i]
					}
					if !bytes.Equal(log.CommittedValues[i], val) {
						fmt.Println("Log not equal is", port)
						//This should never happen
						fmt.Println("POOOOOOOOP")
						return errors.New("Paxos server " + strconv.Itoa(port) + " logs are not correct")
					}
				}
			}

			ps.CommittedMsgs[ps.RoundID] = val
			ps.CommittedMsgsFile.Write(bytes.Trim(val, "\x00"))
			fmt.Println(ps.RoundID, ps.Port, ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
			ps.RoundID++
		}
	}

	return nil
}

func (ps *paxosServer) HandleRecover(args *RecoverArgs, reply *RecoverReplyArgs) error {
	//If this server is behind the server trying to recover then first recover yourself
	if(Active) {
		if args.RoundID > ps.RoundID {
			err := ps.SendRecover(&ManualRecoverArgs{}, &ManualRecoverReply{})
			if err != nil {
				errors.New("Couldn't recover this one so cant send updated logs")
			}
		}

		reply.RoundID = ps.RoundID
		reply.CommittedValues = ps.CommittedMsgs
	} else {
		return errors.New("Server is sleeping")
	}
	return nil
}

//Functions related to Proposer
func (ps *paxosServer) Propose(args *SendMessageArgs, _ *SendMessageReplyArgs) error {
	fmt.Println("In Propose", ps.Port)
	b, err := ps.CheckKill(&args.Tester, "sendPropose", "start")
	if b {
		return err
	}
	fmt.Println("Done with check kill")

	if ps.MaxSeenProposalID > ps.ProposalID {
		//update the proposal id to the now highest seen proposal id
		ps.ProposalID = ps.MaxSeenProposalID + 1
	} else {
		ps.ProposalID++
	}

	majority := ps.NumNodes/2 + ps.NumNodes%2
	//initialize with lowest possible proposal num
	maxPair := &KeyValuePair{-1, nil}

	if len(ps.RPCConnections) == 0{
		err := ps.CreatePaxosConnections()
		if err != nil {
			fmt.Println("Error Occurred", err)
		}
	}

	for port, conn := range ps.RPCConnections  {
		fmt.Println("Sending Propose messages", port)
		proposeArgs := &ProposeArgs{
			RoundID:    ps.RoundID,
			ProposalID: ps.ProposalID,
			Proposer:   ps.Port,
		}
		proposeReply := &ProposeReplyArgs{}

		call := conn.Go("PaxosServer.HandleProposeRequest", proposeArgs, proposeReply, nil)
		timer := time.NewTimer(time.Second*2)

		select {
		case replyCall := <- call.Done:

			if replyCall.Error != nil {
				fmt.Println("Error while calling HandleProposeRequest", replyCall.Error)
				continue
			} else {
				fmt.Println("Proposing reply port", proposeReply.AcceptorPort, "round ", proposeReply.RoundID, "accepted? ",
					proposeReply.Accepted, "Pair is ", proposeReply.Pair, "Proposer is", ps.Port)

				if time.Now().UnixNano()%2 == 0 {
					b, e := ps.CheckKill(&args.Tester, "sendPropose", "mid")
					if b {
						return e
					}
				}

				if proposeReply.Pair != nil {
					if proposeReply.Pair.ProposalID > ps.MaxSeenProposalID {
						ps.MaxSeenProposalID = proposeReply.Pair.ProposalID
					}
				}

				if proposeReply.RoundID > ps.RoundID {
					fmt.Println(ps.Port, "Propose Round Id's do not match need to recover my round id:",ps.RoundID, "proposeReply id", proposeReply.RoundID)
					err := ps.SendRecover(&ManualRecoverArgs{}, &ManualRecoverReply{})
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
		case _ = <-timer.C:
			fmt.Println("RPC Call timed out", ps.Port, "Call to", port)
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
	b, e := ps.CheckKill(tester, "sendAccept", "start")
	if b {
		return e
	}

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

	b, e = ps.CheckKill(tester, "sendAccept", "end")
	if b {
		return e
	}
	if ps.AcceptedQueue.Len() >= majority {
		err := ps.SendCommit(acceptors, value, tester)
		return err
	}
	return errors.New("Couldn't reach a majority to send the accept requests")
}

func (ps *paxosServer) SendCommit(acceptors *list.List, value []byte, tester *Tester) error {
	b, e := ps.CheckKill(tester, "sendCommit", "start")
	if b {
		return e
	}
	fmt.Println("Send Commit")
	ImAnAcceptor := false

	fmt.Println(acceptors.Len())
	for e := acceptors.Front(); e != nil; e = e.Next() {
		reply := e.Value.(*ProposeReplyArgs)

		if reply.AcceptorPort == ps.Port {
			ImAnAcceptor = true
			continue
		}

		conn := ps.RPCConnections[reply.AcceptorPort]

		if time.Now().UnixNano()%2 == 0 {
			b, _ := ps.CheckKill(tester, "sendCommit", "mid")
			if b {
				fmt.Println("SLEEEPING NOW")
				return nil
			}
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
	b, e = ps.CheckKill(tester, "sendCommit", "end")
	if b {
		return e
	}
	fmt.Println("end of send commit ")
	return nil
}

//Functions related to Acceptor
func (ps *paxosServer) HandleProposeRequest(args *ProposeArgs, reply *ProposeReplyArgs) error {
	if(Active) {
		fmt.Println("Handle Propose Request", ps.Port)
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
			fmt.Println(ps.Port, "Trying ot recover", "my port", ps.RoundID, "the args round id is", args.RoundID)
			err := ps.SendRecover(&ManualRecoverArgs{}, &ManualRecoverReply{})
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
	} else {
		reply.RoundID = ps.RoundID
		reply.Accepted = false
		return errors.New("Server is Sleeping")
	}
	return nil
}

func (ps *paxosServer) HandleAcceptRequest(args *AcceptRequestArgs, reply *AcceptReplyArgs) error {
	if(Active) {
		fmt.Println("Handle Accept Request")
		reply.RoundID = ps.RoundID
		reply.AcceptorPort = ps.Port

		if args.RoundID < ps.RoundID {
			reply.Accepted = false
			return nil
		} else if args.RoundID > ps.RoundID {
			fmt.Println("Sending Recover in Handle Accept Request", ps.Port)
			err := ps.SendRecover(&ManualRecoverArgs{}, &ManualRecoverReply{})
			if err != nil {
				fmt.Println("Couldn't recover", err)
			}
		} else if ps.MaxPromisedID >= args.ProposalID {
			reply.Accepted = false
			return nil
		}

		ps.ToCommitQueue.PushBack(KeyValuePair{args.ProposalID, args.Value})
		reply.Accepted = true
	} else {
		return errors.New("Server is sleeping")
	}
	return nil

}

func (ps *paxosServer) HandleCommit(args *CommitArgs, _ *CommitReplyArgs) error {
	if(Active) {
		fmt.Println("Handle commit message paxos server", ps.Port)

		ps.CommittedMsgs[args.RoundID] = args.Value

		trimMsg := strings.Trim(string(args.Value), "\000")
		_, err := ps.CommittedMsgsFile.WriteString(trimMsg)

		if (err != nil) {
			return err
		}

		if ps.RoundID < args.RoundID {
			fmt.Println("Calling Send Recover in Handle Commit", ps.Port)
			err := ps.SendRecover(&ManualRecoverArgs{}, &ManualRecoverReply{})
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
	} else {
		fmt.Println("Server is sleeping")
	}
	return nil
}

func (ps *paxosServer) GetServers(_ *GetServersArgs, reply *GetServersReply) error {
	if Active {

		if ps.NumConnected == ps.NumNodes {
			reply.Servers = ps.Servers
			fmt.Println("number of nodes connected are ", ps.NumConnected)
			reply.Ready = true
		} else {
			fmt.Println("number of nodes connected are ", ps.NumConnected)
			reply.Ready = false

		}
	} else {
		errors.New("Server is sleeping")
	}
	return nil
}

func (ps *paxosServer) ReplaceServer(args *ReplaceServerArgs, reply *ReplaceServerReply) error {
	Active = false
	ps.ProposeAcceptedQueue = list.New()
	ps.AcceptedQueue = list.New()
	ps.ToCommitQueue = list.New()
	fmt.Println("In REPLACE SERVER", ps.NumConnected)
	ps.NumConnected = ps.NumConnected -1

	for i:=0; i<ps.NumNodes; i++ {
		if args.DeadNode == ps.Servers[i] {
			fmt.Println("Deleting deadnode")
			ps.Servers = append(ps.Servers[:i], ps.Servers[i+1:]...)
			break
		}
	}

	fmt.Println("Deleting the rpc connection to the deadnode")
	delete(ps.RPCConnections, args.DeadNode)

	//wait till the new Node joins, same as in newPaxos server
	for {
		fmt.Println("WAITING FOR REGISTER ALL")
		registerArgs := &RegisterArgs{ps.Port}
		reply := &RegisterReplyArgs{}
		var err error

		if args.MasterHostPort == "" {
			//This is a master paxos server that others will register to
			err = ps.RegisterServer(registerArgs, reply)
			if err != nil {
				//fmt.Println(paxosServer.Port, err, "SLEEPING for SECOND")
				time.Sleep(time.Second)
				continue
			}
		} else {
			regular, errDial := rpc.DialHTTP("tcp", "localhost:"+args.MasterHostPort)
			if errDial != nil {
				//fmt.Println("Slave couldn't connect to master", errDial)
				return errDial
			}

			//This is a regular paxos server
			err := regular.Call("PaxosServer.RegisterServer", registerArgs, reply)
			if err != nil {
				//fmt.Println(paxosServer.Port, err, "SLEEPING for SECOND")
				time.Sleep(time.Second)
				continue
			}
		}

		//fmt.Println("Setting paxos servers", paxosServer.Port, reply.NumConnected)
		ps.Servers = reply.Servers
		break
	}

	fmt.Println("DONE REGISTERING")

	serverConn, dialErr := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(args.NewNode))
	if dialErr != nil {
		fmt.Println("Error occured while dialing to all servers", dialErr)
		return dialErr
	} else {
		ps.RPCConnections[args.NewNode] = serverConn
		Active = true
	}

	return nil
}

