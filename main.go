package main

import (
	"flag"
	"log"
	"strings"

	"github.com/degdb/degdb/core"
	"github.com/degdb/degdb/old"
	"github.com/dustin/go-humanize"
)

var useNewDegDB = flag.Bool("new", false, "use the newer version of degdb")
var bindPort = flag.Int("port", 7946, "The port to bind on.")
var initialPeers = flag.String("peers", "", "CSV list of peers to connect to")
var diskAllowed = flag.String("disk", "1G", "amount of disk space to allocate")

func main() {
	flag.Parse()

	var peers []string
	if len(*initialPeers) > 0 {
		peers = strings.Split(*initialPeers, ",")
	}

	diskFloat, _, err := humanize.ParseSI(*diskAllowed)
	if err != nil {
		log.Fatal(err)
	}
	disk := int(diskFloat)

	if *useNewDegDB {
		core.Main(*bindPort, peers, disk)
	} else {
		old.Main(*bindPort)
	}
}
