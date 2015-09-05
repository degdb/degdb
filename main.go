package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"strings"
	"sync"
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

func sqlToTriples(rows *sql.Rows) ([]*Triple, error) {
	defer rows.Close()
	var triples []*Triple
	for rows.Next() {
		var subj, pred, obj string
		if err := rows.Scan(&subj, &pred, &obj); err != nil {
			return nil, err
		}
		triples = append(triples, &Triple{subj, pred, obj})
	}
	return triples, nil
}

func getAllTriples() ([]*Triple, error) {
	rows, err := db.Query("SELECT subj, pred, obj from triples")
	if err != nil {
		return nil, err
	}
	return sqlToTriples(rows)
}

func insertTriple(triple *Triple) error {
	_, err := tripleQuery.Exec(triple.Subj, triple.Pred, triple.Obj, time.Now())
	return err
}

type delegate struct {
	nodes *memberlist.Memberlist
}

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
		err := proto.Unmarshal(msg, &req)
		if err != nil {
			log.Printf("err unmarshalling: %s", err.Error())
			return
		}
		for _, triple := range req.Triples {
			err := insertTriple(triple)
			if err != nil {
				log.Printf("err inserting: %s", err.Error())
				return
			}
		}
	case "Query":
		var query Query
		err := proto.Unmarshal(msg, &query)
		if err != nil {
			log.Printf("err unmarshalling: %s", err.Error())
			return
		}
		sender := d.senderNode(query.Source)
		if sender == nil {
			log.Printf("err can't find sender node: %s", query.Source.Name)
			return
		}
		triples, err := executeQuery(&query)
		if err != nil {
			log.Printf("err executing query: %s", err.Error())
			return
		}
		resp := &QueryResp{
			Source:  server,
			Id:      query.Id,
			Triples: triples,
		}
		msg, err := serializeProto(resp)
		if err != nil {
			log.Printf("err serializing proto: %s", err.Error())
			return
		}
		d.nodes.SendToTCP(sender, msg)
	case "QueryResp":
		var resp QueryResp
		err := proto.Unmarshal(msg, &resp)
		if err != nil {
			log.Printf("err unmarshalling: %s", err.Error())
			return
		}
		req, ok := requestIndex[resp.Id]
		if !ok {
			// ID invalid/old
			log.Printf("err invalid/old request id: %d", resp.Id)
			return
		}
		req.in <- resp.Triples
	}
}

func (d *delegate) senderNode(node *Node) *memberlist.Node {
	for _, n := range d.nodes.Members() {
		if n.Name == node.Name {
			return n
		}
	}
	return nil
}

func serializeProto(msg proto.Message) ([]byte, error) {
	msgI := msg.(interface{})
	messageType := reflect.ValueOf(msgI).Elem().Type().Name()
	out, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append([]byte(messageType+":"), out...), nil
}

var queuedBroadcasts [][]byte
var broadcastLock sync.Mutex

func queueBroadcast(broadcast []byte) {
	broadcastLock.Lock()
	defer broadcastLock.Unlock()

	queuedBroadcasts = append(queuedBroadcasts, broadcast)
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	broadcastLock.Lock()
	defer broadcastLock.Unlock()
	broadcasts := queuedBroadcasts
	queuedBroadcasts = nil
	return broadcasts
}
func (d *delegate) LocalState(join bool) []byte            { return nil }
func (d *delegate) MergeRemoteState(buf []byte, join bool) {}

func astToCallExpr(q ast.Expr) ([]*ast.CallExpr, error) {
	switch e := q.(type) {
	case *ast.CallExpr:
		exprs, err := astToCallExpr(e.Fun)
		if err != nil {
			return nil, err
		}
		exprs[len(exprs)-1].Args = e.Args
		return exprs, err
	case *ast.SelectorExpr:
		f := &ast.CallExpr{Fun: e.Sel}
		exprs, err := astToCallExpr(e.X)
		if err != nil {
			return nil, err

		}
		return append(exprs, f), nil
	case *ast.Ident:
		return []*ast.CallExpr{&ast.CallExpr{Fun: e}}, nil
	default:
		return nil, fmt.Errorf("unknown ast %#v", e)
	}
}

func astToQuery(q ast.Expr) ([]*Query, error) {
	expr, err := astToCallExpr(q)
	if err != nil {
		return nil, err
	}
	var queries []*Query
	for _, e := range expr {
		typ := e.Fun.(*ast.Ident).Name
		switch typ {
		case "Id":
			if len(e.Args) != 1 {
				return nil, fmt.Errorf("Id requires 1 argument not %d", len(e.Args))
			}
			ident, ok := e.Args[0].(*ast.BasicLit)
			if !ok || ident.Kind != token.STRING {
				return nil, fmt.Errorf("Id requires string literal not %#v", e.Args[0])
			}
			queries = append(queries, &Query{Subj: []string{ident.Value[1 : len(ident.Value)-1]}})
		case "Preds":
			query := &Query{}
			if len(e.Args) == 0 {
				return nil, fmt.Errorf("Preds requires at least 1 argument")
			}
			for _, arg := range e.Args {
				ident, ok := arg.(*ast.BasicLit)
				if !ok || ident.Kind != token.STRING {
					return nil, fmt.Errorf("Preds requires string literal not %#v", arg)
				}
				/*
					if ident.Op != token.EQL && ident.Open != token.NEQ {
						return nil, fmt.Errorf("Preds only supports == & !=", e.Args[0])
					}
				*/
				query.Filter = append(query.Filter, &Filter{
					Pred: ident.Value[1 : len(ident.Value)-1],
					Type: Filter_EXISTS,
				})
			}
			queries = append(queries, query)

		default:
			return nil, fmt.Errorf("unknown function %#v", typ)
		}
	}
	return queries, nil
}

type request struct {
	out, in    chan []*Triple
	queries    []*Query
	queryIndex int
	currentID  int64
}

var requestIndex = make(map[int64]*request)

func (r *request) runQuery() error {
	query := r.queries[r.queryIndex]
	if len(query.Filter) == 0 {
		r.queryIndex++
		r.queries[r.queryIndex].Subj = query.Subj
		return r.runQuery()
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(1 * time.Second)
		timeout <- true
	}()

	delete(requestIndex, r.currentID)
	r.currentID = rand.Int63()
	requestIndex[r.currentID] = r

	query.Id = r.currentID
	query.Source = server

	msg, err := serializeProto(query)
	if err != nil {
		return err
	}
	queueBroadcast(msg)
	triples, err := executeQuery(query)
	if err != nil {
		return err
	}

	log.Printf("Triples %#v", triples)

	ch := make(chan []*Triple, 1)
	r.in = ch
	for {
		select {
		case trips := <-ch:
			triples = append(triples, trips...)
		case <-timeout:
			r.queryIndex++
			if len(r.queries) <= r.queryIndex {
				r.out <- triples
			} else {
				query = r.queries[r.queryIndex]
				for _, triple := range triples {
					query.Subj = append(query.Subj, triple.Obj)
				}
				return r.runQuery()
			}
			return fmt.Errorf("external nodes timed out")
		}
	}
}

func executeQuery(q *Query) ([]*Triple, error) {
	var args []interface{}
	sql := "SELECT subj, pred, obj FROM triples WHERE "
	var wheres, filters, ids []string
	for _, id := range q.Subj {
		ids = append(ids, "subj=?")
		args = append(args, id)
	}
	wheres = append(wheres, "("+strings.Join(ids, " OR ")+")")
	for _, filter := range q.Filter {
		switch filter.Type {
		case Filter_EXISTS:
			filters = append(filters, "pred=?")
			args = append(args, filter.Pred)
		case Filter_EQUAL:
			filters = append(filters, "(pred=? AND obj=?)")
			args = append(args, filter.Pred)
			args = append(args, filter.Obj)
		case Filter_NOT_EQUAL:
			filters = append(filters, "(pred=? AND obj!=?)")
			args = append(args, filter.Pred)
			args = append(args, filter.Obj)
		}
	}
	wheres = append(wheres, "("+strings.Join(filters, " OR ")+")")

	sql += strings.Join(wheres, " AND ")
	log.Printf("QUERY: %s %#v", sql, args)
	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	return sqlToTriples(rows)
}

var server *Node

func main() {
	flag.Parse()

	var err error

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
	del.nodes = list

	// Configure the database.
	db, err = sql.Open("sqlite3", "./deg-"+list.LocalNode().Name+".db")
	if err != nil {
		log.Fatal(err)
	}
	if err = setupDB(db); err != nil {
		log.Fatal(err)
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

	ln := list.LocalNode()
	server = &Node{Name: ln.Name}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "file not found ", r.URL.String())
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		q := r.FormValue("q")
		log.Printf("Query: %s", q)
		expr, err := parser.ParseExpr(q)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		calls, err := astToQuery(expr)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		log.Printf("AST %#v", calls)
		ch := make(chan []*Triple, 1)
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(60 * time.Second)
			timeout <- true
		}()
		req := &request{
			out:     ch,
			queries: calls,
		}
		go func() {
			if err := req.runQuery(); err != nil {
				log.Printf("Query ERR %s", err.Error())
			}
		}()
		select {
		case triples := <-ch:
			json.NewEncoder(w).Encode(triples)
		case <-timeout:
			http.Error(w, "query timed out", 500)
		}
	})
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
		packaged, err := serializeProto(req)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
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
