// Package triplestore provides utilities for saving and querying triples with
// a sqlite3 backend.
package triplestore

import (
	"os"

	"github.com/d4l3k/go-disk-usage/du"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"

	"github.com/degdb/degdb/protocol"
)

type TripleStore struct {
	db     gorm.DB
	dbFile string
}

// NewTripleStore returns a TripleStore with the specified file.
func NewTripleStore(file string) (*TripleStore, error) {
	ts := &TripleStore{
		dbFile: file,
	}
	var err error
	if ts.db, err = gorm.Open("sqlite3", file); err != nil {
		return nil, err
	}
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
	var results []*protocol.Triple
	if err := ts.db.Where(*query).Limit(limit).Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// Insert saves a bunch of triples and returns the number asserted.
func (ts *TripleStore) Insert(triples []*protocol.Triple) int {
	count := 0
	for _, triple := range triples {
		if err := ts.db.Create(triple).Error; err != nil {
			continue
		}
		count++
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
