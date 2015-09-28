package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/degdb/degdb/core"
	"github.com/degdb/degdb/old"
	"github.com/dustin/go-humanize"
)

var (
	useNewDegDB  = flag.Bool("new", false, "Use the newer version of degdb.")
	bindPort     = flag.Int("port", 7946, "The port to bind on.")
	initialPeers = flag.String("peers", "", "CSV list of peers to connect to.")
	diskAllowed  = flag.String("disk", "1G", "Amount of disk space to allocate.")
	nodes        = flag.Int("nodes", 1, "Number of nodes to launch in this binary. Development use only. Disables external connections.")
)

func main() {
	flag.Parse()

	var peers []string
	if len(*initialPeers) > 0 && *nodes == 1 {
		peers = strings.Split(*initialPeers, ",")
	}

	diskFloat, _, err := humanize.ParseSI(*diskAllowed)
	if err != nil {
		log.Fatal(err)
	}
	disk := int(diskFloat)

	if *useNewDegDB {
		for i := 0; i < *nodes; i++ {
			port := *bindPort + i
			launchPeers := peers
			launch := func() {
				core.Main(port, launchPeers, disk)
			}
			if i == (*nodes - 1) {
				launch()
			} else {
				go launch()
			}
			peers = []string{fmt.Sprintf("localhost:%d", port)}
		}
	} else {
		old.Main(*bindPort)
	}
}
