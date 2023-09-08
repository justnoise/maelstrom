package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/justnoise/maelstrom/messages"
)

type Server struct {
	nodeID    string
	nextMsgID int
}

func (s *Server) reply(request messages.Request) {
	s.nextMsgID += 1
	reply := messages.Reply{
		Src:  s.nodeID,
		Dest: request.Src,
	}
	switch request.Body.Type {
	case "init":
		reply.Body = &messages.ReplyBodyBase{
			Type:      request.Body.Type + "_ok",
			MsgID:     s.nextMsgID,
			InReplyTo: request.Body.MsgID,
		}
	case "echo":
		reply.Body = &messages.ReplyBodyEcho{
			ReplyBodyBase: messages.ReplyBodyBase{
				Type:      request.Body.Type + "_ok",
				MsgID:     s.nextMsgID,
				InReplyTo: request.Body.MsgID,
			},
			Echo: request.Body.Echo,
		}
	}
	replyJSON, err := json.Marshal(reply)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Sending %s\n", replyJSON)
	fmt.Println(string(replyJSON))
}

func (s *Server) run() {
	for {
		reader := bufio.NewReader(os.Stdin)
		msgJSON, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "Received %s", msgJSON)
		msg := messages.Request{}
		json.Unmarshal([]byte(msgJSON), &msg)
		switch msg.Body.Type {
		case "init":
			s.nodeID = msg.Body.NodeID
			fmt.Fprintf(os.Stderr, "Initialized node %s\n", s.nodeID)
			s.reply(msg)
		case "echo":
			fmt.Fprintf(os.Stderr, "Echoing%+v\n", msg.Body)
			s.reply(msg)
		}
	}
}

func main() {
	s := Server{}
	s.run()
}
