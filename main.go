package main

import (
	"flag"
	"log"

	"github.com/hashicorp/memberlist"
)

var peerAddr = flag.String("peer", "", "The peer address to bootstrap off.")

func main() {
	list, err := memberlist.Create(memberlist.DefaultLocalConfig())
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	// Connect to peers if found.
	if *peerAddr != "" {
		n, err := list.Join([]string{*peerAddr})
		if err != nil {
			panic("Failed to join cluster: " + err.Error())
		}
		log.Printf("Found %d peer nodes.", n)
	}

	for _, member := range list.Members() {
		log.Printf("Node: %s %s\n", member.Name, member.Addr)
	}
}
