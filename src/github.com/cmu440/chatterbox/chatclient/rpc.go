//todo fix this.

package chatclient
import "github.com/cmu440/chatterbox/multipaxos"

type RemoteChatClient interface {
	SendMessage(args *multipaxos.SendMessageArgs, reply *multipaxos.SendMessageReplyArgs) error
	GetServers(args *multipaxos.GetServersArgs, reply*multipaxos.GetServersReply) error
	SendReplaceMessage(args *multipaxos.ReplaceServerArgs, reply *multipaxos.ReplaceServerReply) error
	SendManualRecover(args *multipaxos.ManualRecoverArgs, reply *multipaxos.ManualRecoverReply) error
}

type ChatServer struct {
	// Embed all methods into the struct. See the Effective Go section about
	// embedding for more details: golang.org/doc/effective_go.html#embedding
	RemoteChatClient
}

// Wrap wraps s in a type-safe wrapper struct to ensure that only the desired
// StorageServer methods are exported to receive RPCs.
func Wrap(s RemoteChatClient) RemoteChatClient {
	return &ChatServer{s}
}
