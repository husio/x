package main

import (
	"flag"
	"log"
	"net"
	"strings"

	"github.com/husio/x/hermes/server"
)

func main() {
	var (
		addrFl   = flag.String("addr", "localhost:10001", "Server listening address")
		followFl = flag.String("follow", "", "List of coma separated server addresses to follow")
	)
	flag.Parse()

	ln, err := net.Listen("tcp", *addrFl)
	if err != nil {
		log.Fatalf("cannot start server: %s", err)
	}
	node := server.NewNode()
	for _, addr := range strings.Split(*followFl, ",") {
		if err := node.FollowAddr(addr); err != nil {
			// failing to follow is not critical
			log.Printf("cannot follow %q: %s", addr, err)
		}
	}
	if err := node.Serve(ln); err != nil {
		log.Fatalf("node error: %s", err)
	}
}
