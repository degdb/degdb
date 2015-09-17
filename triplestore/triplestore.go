package triplestore

import (
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"

	"github.com/degdb/degdb/protocol"
)

func Init(file string) error {
	db, err := gorm.Open("sqlite3", file)
	if err != nil {
		return err
	}
	db.CreateTable(&protocol.Triple{})
	db.Model(&protocol.Triple{}).AddIndex("idx_subj", "subj")
	db.Model(&protocol.Triple{}).AddIndex("idx_pred", "pred")
	db.AutoMigrate(&protocol.Triple{})
	return nil
}
