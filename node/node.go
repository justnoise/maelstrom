package node

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/justnoise/maelstrom/messages"
)

type HandlerFunc func(*messages.Request) error

type Set map[interface{}]struct{}

type Node struct {
	sync.Mutex
	logMutex  sync.Mutex
	ID        string
	NodeIDs   []string
	NextMsgID int
	neighbors []string
	Handlers  map[string]HandlerFunc
	messages  Set
}

func New() *Node {
	n := &Node{
		NextMsgID: 0,
		Handlers:  make(map[string]HandlerFunc),
		messages:  make(Set),
	}
	n.Handlers["init"] = n.HandleInit
	n.Handlers["topology"] = n.HandleTopology
	n.Handlers["read"] = n.HandleRead
	n.Handlers["broadcast"] = n.HandleBroadcast
	return n
}

func (n *Node) HandleInit(req *messages.Request) error {
	n.ID = req.Body.NodeID
	n.NodeIDs = req.Body.NodeIDs
	n.Reply(req, &messages.ReplyBase{
		Type: "init_ok",
	})
	n.Log(fmt.Sprintf("Node %s initialized", n.ID))
	return nil
}

func (n *Node) HandleTopology(req *messages.Request) error {
	n.neighbors = req.Body.Topology[n.ID]
	n.Log(fmt.Sprintf("My neighbors are %s", n.neighbors))
	n.Reply(req, &messages.ReplyBase{
		Type: "topology_ok",
	})
	return nil
}

func (n *Node) HandleRead(req *messages.Request) error {
	msgs := make([]interface{}, len(n.messages))
	for msg := range n.messages {
		msgs = append(msgs, msg)
	}
	n.Reply(req, &messages.ReplyMessages{
		ReplyBase: messages.ReplyBase{
			Type: "read_ok",
		},
		Messages: msgs,
	})
	return nil
}

func (n *Node) HandleBroadcast(req *messages.Request) error {
	if req.Body.Message == nil {
		fmt.Fprintln(os.Stderr, "No message in broadcast request")
	}
	if _, ok := n.messages[req.Body.Message]; !ok {
		n.messages[req.Body.Message] = struct{}{}
		for _, neighbor := range n.neighbors {
			n.Send(&messages.Request{
				Src:  n.ID,
				Dest: neighbor,
				Body: messages.RequestBody{
					Type:    "broadcast",
					Message: req.Body.Message,
				},
			})
		}
	}
	if req.Body.MsgID != 0 {
		n.Reply(req, &messages.ReplyBase{
			Type: "broadcast_ok",
		})
	}
	return nil
}

func (n *Node) Log(msg string) {
	n.logMutex.Lock()
	defer n.logMutex.Unlock()
	fmt.Fprintln(os.Stderr, msg)
}

func (n *Node) Reply(req *messages.Request, body messages.ReplyBodyable) {
	body.SetInReplyTo(req.Body.MsgID)
	n.Send(&messages.Reply{
		Src:  n.ID,
		Dest: req.Src,
		Body: body,
	})
}

func (n *Node) Send(msg interface{}) {
	// n.NextMsgID++
	// body.SetMsgID(n.NextMsgID)
	replyJSON, err := json.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}
	n.Log(fmt.Sprintf("Sending %s", replyJSON))
	fmt.Println(string(replyJSON))
}

func (n *Node) ParseMsg(line string) *messages.Request {
	req := &messages.Request{}
	err := json.Unmarshal([]byte(line), req)
	if err != nil {
		log.Fatal(err)
	}
	return req
}

func (n *Node) Run() {
	for {
		reader := bufio.NewReader(os.Stdin)
		msgJSON, err := reader.ReadString('\n')
		n.Log(fmt.Sprintf("Received %s", msgJSON))
		msg := n.ParseMsg(msgJSON)

		handler, ok := n.Handlers[msg.Body.Type]
		if !ok {
			n.Log(fmt.Sprintf("No handler for %s", msg.Body.Type))
			os.Exit(1)
		}
		n.Lock()
		err = handler(msg)
		n.Unlock()
		if err != nil {
			n.Log(fmt.Sprintf("Error handling %+v: %s", msg, err))
		}
	}
}
