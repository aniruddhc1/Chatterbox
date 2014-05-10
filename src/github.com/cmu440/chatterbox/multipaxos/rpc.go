package multipaxos

type RemotePaxosServer interface {
	HandleRecover(args *RecoverArgs, reply *RecoverReplyArgs) error
	RegisterServer(args *RegisterArgs, reply *RegisterReplyArgs) error
	SendMessage(args *SendMessageArgs, reply *SendMessageReplyArgs) error
	HandleProposeRequest(args *ProposeArgs, reply *ProposeReplyArgs) error
	HandleAcceptRequest(args *AcceptRequestArgs, reply *AcceptReplyArgs) error
	HandleCommit(args *CommitArgs, reply *CommitReplyArgs) error
	GetServers(args *GetServersArgs, reply *GetServersReply) error
	ServeMessageFile(args *FileArgs, reply *FileReply) error
	WakeupServer(args *WakeupRequestArgs, reply *WakeupReplyArgs) error
	ReplaceServer(args *ReplaceServerArgs, reply *ReplaceServerReply) error
	SendRecover(args *ManualRecoverArgs, reply *ManualRecoverReply) error
}

type PaxosServer struct {
// Embed all methods into the struct. See the Effective Go section about
// embedding for more details: golang.org/doc/effective_go.html#embedding
	RemotePaxosServer
}

// Wrap wraps s in a type-safe wrapper struct to ensure that only the desired
// StorageServer methods are exported to receive RPCs.
func Wrap(s RemotePaxosServer) RemotePaxosServer {
	return &PaxosServer{s}
}
