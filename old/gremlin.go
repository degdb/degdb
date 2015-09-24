package old

import (
	"fmt"
	"log"
	"time"

	"github.com/google/cayley/graph"
	"github.com/google/cayley/quad"
	"github.com/google/cayley/query"
	"github.com/google/cayley/query/gremlin"
	"github.com/google/cayley/query/mql"
)

type QuadStore struct{}

// The only way in is through building a transaction, which
// is done by a replication strategy.
func (qs *QuadStore) ApplyDeltas(d []graph.Delta, o graph.IgnoreOpts) error {
	log.Printf("ApplyDeltas(%#v, %#v)", d, o)
	return nil
}

// Given an opaque token, returns the quad for that token from the store.
func (qs *QuadStore) Quad(v graph.Value) quad.Quad {
	log.Printf("Quad(%#v)", v)
	return quad.Quad{}
}

// Given a direction and a token, creates an iterator of links which have
// that node token in that directional field.
func (qs *QuadStore) QuadIterator(d quad.Direction, v graph.Value) graph.Iterator {
	log.Printf("QuadIterator(%#v, %#v)", d, v)
	return nil
}

// Returns an iterator enumerating all nodes in the graph.
func (qs *QuadStore) NodesAllIterator() graph.Iterator {
	log.Printf("NodesAllIterator()")
	return nil
}

// Returns an iterator enumerating all links in the graph.
func (qs *QuadStore) QuadsAllIterator() graph.Iterator {
	log.Printf("QuadsAllIterator()")
	return nil
}

// Given a node ID, return the opaque token used by the QuadStore
// to represent that id.
func (qs *QuadStore) ValueOf(str string) graph.Value {
	log.Printf("ValueOf(%#v)", str)
	return str
}

// Given an opaque token, return the node that it represents.
func (qs *QuadStore) NameOf(val graph.Value) string {
	switch v := val.(type) {
	case string:
		return v
	default:
		log.Printf("Unknown value type: %#v", v)
	}
	return ""
}

// Returns the number of quads currently stored.
func (qs *QuadStore) Size() int64 {
	log.Printf("Size()")
	return 10000
}

// The last replicated transaction ID that this quadstore has verified.
func (qs *QuadStore) Horizon() graph.PrimaryKey {
	log.Printf("Horizon()")
	return graph.PrimaryKey{}
}

// Creates a fixed iterator which can compare Values
func (qs *QuadStore) FixedIterator() graph.FixedIterator {
	log.Printf("FixedIterator()")
	return nil
}

// Optimize an iterator in the context of the quad store.
// Suppose we have a better index for the passed tree; this
// gives the QuadStore the opportunity to replace it
// with a more efficient iterator.
func (qs *QuadStore) OptimizeIterator(it graph.Iterator) (graph.Iterator, bool) { return it, false }

// Close the quad store and clean up. (Flush to disk, cleanly
// sever connections, etc)
func (qs *QuadStore) Close() {}

// Convenience function for speed. Given a quad token and a direction
// return the node token for that direction. Sometimes, a QuadStore
// can do this without going all the way to the backing store, and
// gives the QuadStore the opportunity to make this optimization.
//
// Iterators will call this. At worst, a valid implementation is
//
//  qs.ValueOf(qs.Quad(id).Get(dir))
//
func (qs *QuadStore) QuadDirection(id graph.Value, d quad.Direction) graph.Value {
	return qs.ValueOf(qs.Quad(id).Get(d))
}

// Get the type of QuadStore
//TODO replace this using reflection
func (qs *QuadStore) Type() string {
	log.Printf("Type()")
	return "main.QuadStore"
}

func runQuery(queryLang, code string) (interface{}, error) {
	qs := &QuadStore{}
	var ses query.HTTP
	switch queryLang {
	case "gremlin":
		ses = gremlin.NewSession(qs, 100*time.Second, false)
	case "mql":
		ses = mql.NewSession(qs)
	default:
	}
	result, err := ses.Parse(code)
	switch result {
	case query.Parsed:
		output, err := Run(code, ses)
		if err != nil {
			return nil, err
		}
		return output, nil
	case query.ParseFail:
		ses = nil
		return nil, err
	default:
		ses = nil
		return nil, fmt.Errorf("Incomplete data?")
	}
}

func Run(q string, ses query.HTTP) (interface{}, error) {
	c := make(chan interface{}, 5)
	log.Println("Executing")
	go ses.Execute(q, c, 100)
	for res := range c {
		ses.Collate(res)
	}
	return ses.Results()
}
