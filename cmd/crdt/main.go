package main

import (
	node "github.com/justnoise/maelstrom/crdt_node"
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
