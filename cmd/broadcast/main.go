package main

import (
	"github.com/justnoise/maelstrom/node"
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
