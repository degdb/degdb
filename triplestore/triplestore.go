// Package triplestore provides utilities for saving and querying triples with
// a sqlite3 backend.
package triplestore

import (
	"log"
	"os"
	"strings"

	"github.com/d4l3k/go-disk-usage/du"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"

	"github.com/degdb/degdb/protocol"
)

const (
	// BloomFalsePositiveRate with 1 million items is about 5.14MB in size. Scales linearly.
	// http://hur.st/bloomfilter?n=1000000&p=1.0E-9
	BloomFalsePositiveRate = 1.0e-9
)

type TripleStore struct {
	db     gorm.DB
	dbFile string
}

// NewTripleStore returns a TripleStore with the specified file.
func NewTripleStore(file string, logger *log.Logger) (*TripleStore, error) {
	ts := &TripleStore{
		dbFile: file,
	}
	var err error
	if ts.db, err = gorm.Open("sqlite3", file); err != nil {
		return nil, err
	}
	ts.db.SetLogger(logger)
	ts.db.CreateTable(&protocol.Triple{})
	ts.db.Model(&protocol.Triple{}).AddIndex("idx_subj", "subj")
	ts.db.Model(&protocol.Triple{}).AddIndex("idx_pred", "pred")
	ts.db.Model(&protocol.Triple{}).AddUniqueIndex("idx_subj_pred_obj", "subj", "pred", "obj")
	ts.db.AutoMigrate(&protocol.Triple{})
	return ts, nil
}

// Query does a WHERE search with the set fields on query. A limit of -1
// returns all results.
func (ts *TripleStore) Query(query *protocol.Triple, limit int) ([]*protocol.Triple, error) {
	dbq := ts.db.Where(*query)
	if limit > 0 {
		dbq = dbq.Limit(limit)
	}
	var results []*protocol.Triple
	if err := dbq.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// QueryArrayOp runs an ArrayOp against the local triple store.
func (ts *TripleStore) QueryArrayOp(q *protocol.ArrayOp, limit int) ([]*protocol.Triple, error) {
	query := ArrayOpToSQL(q)
	args := make([]interface{}, len(query)-1)
	for i, arg := range query[1:] {
		args[i] = arg
	}
	dbq := ts.db.Where(query[0], args...)
	if limit > 0 {
		dbq = dbq.Limit(limit)
	}
	var results []*protocol.Triple
	if err := dbq.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func ArrayOpToSQL(q *protocol.ArrayOp) []string {
	var rules []string
	args := []string{""}
	switch q.Mode {
	case protocol.AND, protocol.OR:
		for _, triple := range q.Triples {
			sql := TripleToSQL(triple)
			args = append(args, sql[1:]...)
			rules = append(rules, sql[0])
		}
		for _, arrayOp := range q.Arguments {
			sql := ArrayOpToSQL(arrayOp)
			args = append(args, sql[1:]...)
			rules = append(rules, sql[0])
		}
		mode := protocol.ArrayOp_Mode_name[int32(q.Mode)]
		args[0] = "(" + strings.Join(rules, ") "+mode+" (") + ")"
	case protocol.NOT:
		if len(q.Triples) > 0 {
			args = TripleToSQL(q.Triples[0])
		} else if len(q.Arguments) > 0 {
			args = ArrayOpToSQL(q.Arguments[0])
		}
		args[0] = "NOT (" + args[0] + ")"
	}
	return args
}

func TripleToSQL(triple *protocol.Triple) []string {
	var rules []string
	args := []string{""}
	if len(triple.Subj) > 0 {
		rules = append(rules, "subj = ?")
		args = append(args, triple.Subj)
	}
	if len(triple.Pred) > 0 {
		rules = append(rules, "pred = ?")
		args = append(args, triple.Pred)
	}
	if len(triple.Obj) > 0 {
		rules = append(rules, "obj = ?")
		args = append(args, triple.Obj)
	}
	if len(triple.Lang) > 0 {
		rules = append(rules, "lang = ?")
		args = append(args, triple.Lang)
	}
	if len(triple.Author) > 0 {
		rules = append(rules, "author = ?")
		args = append(args, triple.Author)
	}
	args[0] = strings.Join(rules, " AND ")
	return args
}

// Insert saves a bunch of triples and returns the number asserted.
func (ts *TripleStore) Insert(triples []*protocol.Triple) int {
	count := 0
	tx := ts.db.Begin()
	for _, triple := range triples {
		if err := tx.Create(triple).Error; err != nil {
			continue
		}
		count++
	}
	if err := tx.Commit().Error; err != nil {
		return 0
	}
	return count
}

// Info represents the state of the database.
type Info struct {
	Triples, DiskSize, AvailableSpace uint64
}

// Size returns an info object about the number of triples and file size of the
// database.
func (ts *TripleStore) Size() (*Info, error) {
	fileInfo, err := os.Stat(ts.dbFile)
	if err != nil {
		return nil, err
	}
	space := du.NewDiskUsage(ts.dbFile)
	i := &Info{
		DiskSize:       uint64(fileInfo.Size()),
		AvailableSpace: space.Available(),
	}
	ts.db.Model(&protocol.Triple{}).Count(&i.Triples)

	return i, nil
}
