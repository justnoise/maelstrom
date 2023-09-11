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

type task struct {
	f  func()
	dt time.Duration
}

type Node struct {
	sync.Mutex
	stderrMutex sync.Mutex
	stdoutMutex sync.Mutex
	ID          string
	NodeIDs     []string
	NextMsgID   int
	neighbors   []string
	Handlers    map[string]HandlerFunc
	tasks       []task
	crdt        CRDT
}

func New() *Node {
	n := &Node{
		NextMsgID: 0,
		Handlers:  make(map[string]HandlerFunc),
		crdt:      NewPNCounter(),
	}
	n.tasks = append(n.tasks, task{f: n.DoReplicate, dt: 1 * time.Second})
	n.Handlers["init"] = n.HandleInit
	n.Handlers["topology"] = n.HandleTopology
	n.Handlers["read"] = n.HandleRead
	n.Handlers["add"] = n.HandleAdd
	n.Handlers["replicate"] = n.HandleReplicate
	return n
}

func (n *Node) DoReplicate() {
	serializedValue, err := json.Marshal(n.crdt)
	if err != nil {
		panic(err)
	}
	for _, nodeID := range n.NodeIDs {
		if n.ID == nodeID {
			continue
		}
		n.Send(&messages.Request{
			Src:  n.ID,
			Dest: nodeID,
			Body: messages.RequestBody{
				Type:  "replicate",
				Value: string(serializedValue),
			},
		})
	}
}

func (n *Node) HandleInit(req messages.Request) error {
	n.Lock()
	n.ID = req.Body.NodeID
	n.NodeIDs = req.Body.NodeIDs
	n.Unlock()
	n.Reply(&req, &messages.ReplyBodyBase{
		Type: "init_ok",
	})
	n.StartPeriodicTasks()
	n.Log(fmt.Sprintf("Node %s initialized", n.ID))
	return nil
}

func (n *Node) StartPeriodicTasks() {
	for _, task := range n.tasks {
		go func() {
			for {
				task.f()
				time.Sleep(task.dt)
			}
		}()
	}
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

func (n *Node) HandleReplicate(req messages.Request) error {
	n.Lock()
	defer n.Unlock()
	v := NewPNCounter()
	err := json.Unmarshal([]byte(req.Body.Value.(string)), &v)
	if err != nil {
		panic(err)
	}
	n.crdt.Merge(v)
	return nil
}

func (n *Node) HandleAdd(req messages.Request) error {
	n.crdt.Add(n.ID, req.Body.Delta)
	n.Reply(&req, &messages.ReplyBodyBase{
		Type: "add_ok",
	})
	return nil
}

func (n *Node) HandleRead(req messages.Request) error {
	n.Reply(&req, &messages.ReplyBodyValue{
		ReplyBodyBase: messages.ReplyBodyBase{
			Type: "read_ok",
		},
		Value: n.crdt.Read(),
	})
	return nil
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
