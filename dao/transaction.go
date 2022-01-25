package dao

import (
	"fmt"

	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Transaction interface {
	Start()
	Save(interface{}) error
	Commit() error
	Rollback()
}

type DbTransaction struct {
	transaction *gorm.DB
}

// Begins and stores a transaction context to be used from the other Transaction
// methods.
func (t *DbTransaction) Start() {
	if t.transaction == nil {
		t.transaction = DB.Begin()
	}
}

/*
	Saves any value to a specified model's table

	Has some extra logic regarding slices - if the slice length is 0 it returns
	nil like nothing happened otherwise gorm reports a false-positive error
	"empty slice found":
	https://github.com/go-gorm/gorm/blob/e5894ca44951fecc3b3f31f1aa46df7de6024b04/errors.go#L33
*/
func (t *DbTransaction) Save(model interface{}, val interface{}) error {
	if t.transaction == nil {
		return fmt.Errorf("transaction not started")
	}

	switch model.(type) {
	case m.Source:
		if sources, ok := val.(*[]m.Source); ok && len(*sources) == 0 {
			return nil
		}

		return t.transaction.Model(&m.Source{}).Omit(clause.Associations).Save(val).Error
	case m.Application:
		if apps, ok := val.(*[]m.Application); ok && len(*apps) == 0 {
			return nil
		}

		return t.transaction.Model(&m.Application{}).Omit(clause.Associations).Save(val).Error
	case m.Endpoint:
		if endpoints, ok := val.(*[]m.Endpoint); ok && len(*endpoints) == 0 {
			return nil
		}

		return t.transaction.Model(&m.Endpoint{}).Omit(clause.Associations).Save(val).Error
	case m.ApplicationAuthentication:
		if appauths, ok := val.(*[]m.ApplicationAuthentication); ok && len(*appauths) == 0 {
			return nil
		}
		return t.transaction.Model(&m.ApplicationAuthentication{}).Omit(clause.Associations).Save(val).Error
	}

	return fmt.Errorf("unsupported transactional type %t", model)
}

// commits the current transcation, returning an error if anything goes wrong
func (t *DbTransaction) Commit() error {
	if t.transaction == nil {
		return fmt.Errorf("transaction not started")
	}

	return t.transaction.Commit().Error
}

// rolls back the current transaction - error is handled within and logged
func (t *DbTransaction) Rollback() {
	if t.transaction == nil {
		l.Log.Errorf("Transaction not started")
		return
	}

	if err := t.transaction.Rollback(); err != nil {
		l.Log.Warnf("Failed to rollback transaction: %v", err)
	}
}
