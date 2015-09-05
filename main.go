package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/memberlist"

	_ "github.com/mattn/go-sqlite3"
)

var peerAddr = flag.String("peer", "", "The peer address to bootstrap off.")

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

var db *sql.DB

type Triple struct {
	Subj, Pred, Obj string
}

func insertTriple(triple *Triple) error {
	_, err := tripleQuery.Exec(triple.Subj, triple.Pred, triple.Obj, time.Now())
	return err
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./deg.db")
	if err != nil {
		log.Fatal(err)
	}
	if err = setupDB(db); err != nil {
		log.Fatal(err)
	}

	list, err := memberlist.Create(memberlist.DefaultWANConfig())
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
		log.Printf("Node: %s %s\n", member.Name, member.Addr)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
