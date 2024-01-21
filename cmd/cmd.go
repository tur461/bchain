package cmd

import (
	"bchain/network"
	"os"
)

func ParseAndRun() {
	p := network.Peer{}
	args := os.Args[1:]

	if args[0] == "--server" {
		p.Setup("server")
	} else if args[0] == "--client" {
		p.Setup("client")
	} else {
		p.Setup(args[0])
	}

}
