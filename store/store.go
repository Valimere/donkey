package store

import "gorm.io/gorm"

type Datastore interface {
	FirstOrCreate(out interface{}, conds ...interface{}) *gorm.DB
	// define other required methods
}
