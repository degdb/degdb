package main

import (
	"flag"

	"github.com/degdb/degdb/core"
	"github.com/degdb/degdb/old"
)

var useNewDegDB = flag.Bool("new", false, "use the newer version of degdb")

func main() {
	flag.Parse()
	if *useNewDegDB {
		core.Main()
	} else {
		old.Main()
	}
}
