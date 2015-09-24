// Package triplestore provides utilities for saving and querying triples with
// a sqlite3 backend.
package triplestore

import (
	"os"
	"sync"

	"github.com/d4l3k/go-disk-usage/du"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"

	"github.com/degdb/degdb/protocol"
)

var db gorm.DB
var dbFile string

var initOnce sync.Once

// Init initalizes the triplestore with the specified file.
func Init(file string) error {
	var err error
	initOnce.Do(func() {
		err = initDB(file)
	})
	return err
}

func initDB(file string) error {
	var err error
	dbFile = file
	db, err = gorm.Open("sqlite3", file)
	if err != nil {
		return err
	}
	db.CreateTable(&protocol.Triple{})
	db.Model(&protocol.Triple{}).AddIndex("idx_subj", "subj")
	db.Model(&protocol.Triple{}).AddIndex("idx_pred", "pred")
	db.Model(&protocol.Triple{}).AddUniqueIndex("idx_subj_pred_obj", "subj", "pred", "obj")
	db.AutoMigrate(&protocol.Triple{})
	return nil
}

// Query does a WHERE search with the set fields on query. A limit of -1
// returns all results.
func Query(query *protocol.Triple, limit int) ([]*protocol.Triple, error) {
	var results []*protocol.Triple
	if err := db.Where(*query).Limit(limit).Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// Insert saves a bunch of triples and returns the number asserted.
func Insert(triples []*protocol.Triple) int {
	count := 0
	for _, triple := range triples {
		if err := db.Create(triple).Error; err != nil {
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
func Size() (*Info, error) {
	fileInfo, err := os.Stat(dbFile)
	if err != nil {
		return nil, err
	}
	space := du.NewDiskUsage(dbFile)
	i := &Info{
		DiskSize:       uint64(fileInfo.Size()),
		AvailableSpace: space.Available(),
	}
	db.Model(&protocol.Triple{}).Count(&i.Triples)

	return i, nil
}
