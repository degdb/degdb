package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"

	_ "github.com/mattn/go-sqlite3"
)

//go:generate protoc --go_out=. main.proto

var peerAddr = flag.String("peer", "", "The peer address to bootstrap off.")
var bindPort = flag.Int("port", 7946, "The port to bind on.")
var bindAddr = flag.String("hostname", "", "The hostname to use.")
var webAddr = flag.String("webAddr", ":8080", "The bin address for the webserver.")

const newDBQuery = `
CREATE TABLE IF NOT EXISTS 'triples' (
	'uid' INTEGER PRIMARY KEY AUTOINCREMENT,
	'subj' TEXT NULL,
	'pred' TEXT NULL,
	'obj' TEXT NULL,
	'created' DATE NULL
)
`

var tripleQuery *sql.Stmt
var db *sql.DB

func setupDB(db *sql.DB) error {
	var err error
	if _, err = db.Exec(newDBQuery); err != nil {
		return err
	}
	tripleQuery, err = db.Prepare("INSERT INTO triples(subj, pred, obj, created) values(?,?,?,?)")
	if err != nil {
		return err
	}
	return nil
}

func getAllTriples() ([]*Triple, error) {
	rows, err := db.Query("SELECT subj, pred, obj from triples")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var triples []*Triple
	for rows.Next() {
		var subj, pred, obj string
		err = rows.Scan(&subj, &pred, &obj)
		if err != nil {
			return nil, err
		}
		triples = append(triples, &Triple{subj, pred, obj})
	}
	return triples, nil
}

func insertTriple(triple *Triple) error {
	_, err := tripleQuery.Exec(triple.Subj, triple.Pred, triple.Obj, time.Now())
	return err
}

type delegate struct{}

func (d *delegate) NodeMeta(limit int) []byte { return nil }
func (d *delegate) NotifyMsg(msg []byte) {
	i := bytes.IndexByte(msg, byte(':'))
	if i == -1 {
		log.Printf("Bad message: %s", msg)
		return
	}
	typ := string(msg[:i])
	msg = msg[i+1:]
	log.Printf("Message (%s): %s", typ, msg)

	switch typ {
	case "AddTriplesRequest":
		var req AddTriplesRequest
		proto.Unmarshal(msg, &req)
		for _, triple := range req.Triples {
			log.Printf("Triple %#v", triple)
			err := insertTriple(triple)
			if err != nil {
				log.Printf("err inserting: %s", err.Error())
			}
		}
	}
}
func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte { return nil }
func (d *delegate) LocalState(join bool) []byte                { return nil }
func (d *delegate) MergeRemoteState(buf []byte, join bool)     {}

func main() {
	flag.Parse()

	var err error
	db, err = sql.Open("sqlite3", "./deg.db")
	if err != nil {
		log.Fatal(err)
	}
	if err = setupDB(db); err != nil {
		log.Fatal(err)
	}

	del := &delegate{}
	config := memberlist.DefaultWANConfig()
	config.Delegate = del
	log.Printf("Listening on %s:%d", config.BindAddr, config.BindPort)
	config.BindPort = *bindPort
	if len(*bindAddr) > 0 {
		config.Name = *bindAddr
	}
	list, err := memberlist.Create(config)
	if err != nil {
		log.Fatal("Failed to create memberlist: " + err.Error())
	}

	// Connect to peers if found.
	if *peerAddr != "" {
		n, err := list.Join([]string{*peerAddr})
		if err != nil {
			log.Fatal("Failed to join cluster: " + err.Error())
		}
		log.Printf("Found %d peer nodes.", n)
	}

	for _, member := range list.Members() {
		log.Printf("Node: %s:%d %s\n", member.Name, member.Port, member.Addr)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {})
	http.HandleFunc("/api/v1/insert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "endpoint needs POST", 400)
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		subj := r.FormValue("subj")
		pred := r.FormValue("pred")
		obj := r.FormValue("obj")
		triple := &Triple{
			subj, pred, obj,
		}
		req := &AddTriplesRequest{
			&Node{},
			[]*Triple{triple},
		}
		members := list.Members()
		node := members[rand.Intn(len(members))]
		out, err := proto.Marshal(req)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		packaged := append([]byte("AddTriplesRequest:"), out...)
		list.SendToTCP(node, packaged)
	})
	http.HandleFunc("/api/v1/triples", func(w http.ResponseWriter, r *http.Request) {
		triples, err := getAllTriples()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(triples)
	})
	http.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "running")
	})

	log.Fatal(http.ListenAndServe(*webAddr, nil))
}
