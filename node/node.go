package node

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/justnoise/maelstrom/messages"
)

type HandlerFunc func(messages.Request) error

type Set map[interface{}]struct{}

type Node struct {
	sync.Mutex
	stderrMutex      sync.Mutex
	stdoutMutex      sync.Mutex
	ID               string
	NodeIDs          []string
	NextMsgID        int
	neighbors        []string
	Handlers         map[string]HandlerFunc
	messages         Set
	inFlightMessages map[int]struct{}
}

func New() *Node {
	n := &Node{
		NextMsgID:        0,
		Handlers:         make(map[string]HandlerFunc),
		messages:         make(Set),
		inFlightMessages: make(map[int]struct{}),
	}
	n.Handlers["init"] = n.HandleInit
	n.Handlers["topology"] = n.HandleTopology
	n.Handlers["read"] = n.HandleRead
	n.Handlers["broadcast"] = n.HandleBroadcast
	n.Handlers["broadcast_ok"] = n.HandleBroadcastOK
	return n
}

func (n *Node) HandleInit(req messages.Request) error {
	n.Lock()
	n.ID = req.Body.NodeID
	n.NodeIDs = req.Body.NodeIDs
	n.Unlock()
	n.Reply(&req, &messages.ReplyBodyBase{
		Type: "init_ok",
	})
	n.Log(fmt.Sprintf("Node %s initialized", n.ID))
	return nil
}

func (n *Node) HandleTopology(req messages.Request) error {
	n.Lock()
	n.neighbors = req.Body.Topology[n.ID]
	n.Unlock()
	n.Log(fmt.Sprintf("My neighbors are %s", n.neighbors))
	n.Reply(&req, &messages.ReplyBodyBase{
		Type: "topology_ok",
	})
	return nil
}

func (n *Node) HandleRead(req messages.Request) error {
	n.Lock()
	msgs := make([]interface{}, len(n.messages))
	for msg := range n.messages {
		msgs = append(msgs, msg)
	}
	n.Unlock()
	n.Reply(&req, &messages.ReplyBodyMessages{
		ReplyBodyBase: messages.ReplyBodyBase{
			Type: "read_ok",
		},
		Messages: msgs,
	})
	return nil
}

func (n *Node) HandleBroadcastOK(req messages.Request) error {
	n.Lock()
	if _, ok := n.inFlightMessages[req.Body.InReplyTo]; !ok {
		n.Log(fmt.Sprintf("Received broadcast_ok for unknown message %d", req.Body.InReplyTo))
	} else {
		delete(n.inFlightMessages, req.Body.InReplyTo)
	}
	n.Unlock()
	return nil
}

func (n *Node) HandleBroadcast(req messages.Request) error {
	if req.Body.Message == nil {
		n.Log("No message in broadcast request")
		return nil
	}
	n.Reply(&req, &messages.ReplyBodyBase{
		Type: "broadcast_ok",
	})
	// If we haven't already seen this message, broadcast it to our neighbors
	n.Lock()
	_, alreadySeenMessage := n.messages[req.Body.Message]
	if !alreadySeenMessage {
		n.messages[req.Body.Message] = struct{}{}
	}
	n.Unlock()
	if !alreadySeenMessage {
		for _, neighbor := range n.neighbors {
			if neighbor == req.Src {
				continue
			}
			go n.RPCWithRetry(neighbor, messages.RequestBody{
				Type:    "broadcast",
				Message: req.Body.Message,
			})
		}
	}
	return nil
}

func (n *Node) RPCWithRetry(dest string, body messages.RequestBody) {
	n.Lock()
	n.NextMsgID++
	msgID := n.NextMsgID
	n.inFlightMessages[msgID] = struct{}{}
	n.Unlock()
	body.MsgID = msgID
	// todo: while we don't have a reply, try to send the message
	messageUnacked := true
	for messageUnacked {
		n.Log(fmt.Sprintf("There are %d unacked messages", len(n.inFlightMessages)))
		n.Send(&messages.Request{
			Src:  n.ID,
			Dest: dest,
			Body: body,
		})
		time.Sleep(1 * time.Second)
		n.Lock()
		_, messageUnacked = n.inFlightMessages[msgID]
		n.Unlock()
	}
}

func (n *Node) Log(msg string) {
	n.stderrMutex.Lock()
	fmt.Fprintln(os.Stderr, msg)
	defer n.stderrMutex.Unlock()
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
	replyJSON, err := json.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}
	n.Log(fmt.Sprintf("Sending %s", replyJSON))
	n.putMsgOnWire(string(replyJSON))
}

func (n *Node) putMsgOnWire(msg string) {
	n.stdoutMutex.Lock()
	fmt.Println(msg)
	n.stdoutMutex.Unlock()
}

func (n *Node) ParseMsg(line string) messages.Request {
	req := messages.Request{}
	err := json.Unmarshal([]byte(line), &req)
	if err != nil {
		n.Log(fmt.Sprintf("Error parsing message: %s into %+v", line, req))
		log.Fatal(err)
	}
	return req
}

func (n *Node) Run() {
	reader := bufio.NewReader(os.Stdin)
	for {
		msgJSON, err := reader.ReadString('\n')
		if err != nil {
			n.Log(fmt.Sprintf("Error reading from stdin: %s", err))
			os.Exit(1)
		}
		n.Log(fmt.Sprintf("Received %s", msgJSON))
		if len(msgJSON) <= 1 {
			n.Log(fmt.Sprintf("Empty message received"))
			continue
		}
		req := n.ParseMsg(msgJSON)
		handler, ok := n.Handlers[req.Body.Type]
		if !ok {
			n.Log(fmt.Sprintf("No handler for %s", req.Body.Type))
			os.Exit(1)
		}
		go func() {
			err = handler(req)
			if err != nil {
				n.Log(fmt.Sprintf("Error handling %+v: %s", req, err))
			}
		}()
	}
}
