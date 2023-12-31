package main

import (
	node "github.com/justnoise/maelstrom/broadcast_node"
)

type Server struct {
	node *node.Node
}

func main() {
	server := Server{
		node: node.New(),
	}
	server.node.Run()
}
