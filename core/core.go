// Package core contains the rewritten degdb code.
package core

import (
	"log"

	"github.com/degdb/degdb/network"
)

func Main() {
	server, err := network.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	server.Listen(8181)
}
