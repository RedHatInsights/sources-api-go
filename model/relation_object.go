package model

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RelationSetting struct {
	RelationType string
	Through      string
}

type RelationObject struct {
	Id              int64
	CurrentTenantID int64
	baseObject      interface{}
	settings        map[string]RelationSetting
}

func (relationObject *RelationObject) HasManyRelation(query *gorm.DB, model interface{}) *gorm.DB {
	expression := []clause.Expression{clause.Eq{
		Column: clause.Column{Table: clause.CurrentTable, Name: relationObject.foreignKeyFrom()},
		Value:  relationObject.Id},
	}

	return query.Clauses(clause.Where{Exprs: expression}).Model(model)
}

func (relationObject *RelationObject) HasMany(model interface{}, query *gorm.DB) *gorm.DB {
	modelName := strcase.ToSnake(reflect.TypeOf(model).Elem().Name())
	if relationObject.settings[modelName].RelationType == "through" {
		return relationObject.HasManyThrough(query, model)
	}

	return relationObject.HasManyRelation(query, model)
}

func (relationObject *RelationObject) SelectStatementFor(query *gorm.DB, model interface{}) string {
	statement := &gorm.Statement{DB: query}
	err := statement.Parse(model)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to parse statement: %v", err))
	}

	var statementFields []string
	for field := range statement.Schema.FieldsByDBName {
		statementFields = append(statementFields, statement.Table+"."+field)
	}

	return strings.Join(statementFields, ", ")
}

func (relationObject *RelationObject) HasManyThrough(query *gorm.DB, model interface{}) *gorm.DB {
	query = query.Debug().Select(relationObject.SelectStatementFor(query, model))
	query.Statement.Distinct = true

	subCollectionModel := strcase.ToSnake(reflect.TypeOf(model).Elem().Name())
	query.Model(model)
	relationSetting := relationObject.settings[subCollectionModel]
	expression := []clause.Expression{clause.Eq{
		Column: clause.Column{Table: clause.CurrentTable, Name: "id"},
		Value:  clause.Column{Table: relationSetting.Through, Name: subCollectionModel + "_id"},
	}}

	if relationObject.CurrentTenantID != 0 {
		expression = append(expression, clause.Eq{Column: clause.Column{Table: relationSetting.Through, Name: "tenant_id"},
			Value: relationObject.CurrentTenantID})
	}

	joins := append([]clause.Join{}, clause.Join{
		Type:  clause.InnerJoin,
		Table: clause.Table{Name: relationSetting.Through},
		ON:    clause.Where{Exprs: expression},
	})

	query.Statement.AddClause(clause.From{Joins: joins})
	query.Where(relationSetting.Through+"."+relationObject.foreignKeyFrom()+" = ?", relationObject.Id)

	return query
}

func (relationObject *RelationObject) foreignKeyFrom() string {
	return strcase.ToSnake(reflect.TypeOf(relationObject.baseObject).Name()) + "_id"
}

func (relationObject *RelationObject) setRelationInfo(query *gorm.DB) error {
	switch object := relationObject.baseObject.(type) {
	case SourceType:
		relationObject.Id = object.Id
		relationObject.settings = object.RelationInfo()
		if query != nil {
			resultPrimaryCollection := query.First(&object)
			if resultPrimaryCollection.Error != nil {
				return resultPrimaryCollection.Error
			}
		}
	case ApplicationType:
		relationObject.Id = object.Id
		relationObject.settings = object.RelationInfo()
		if query != nil {
			resultPrimaryCollection := query.First(&object)
			if resultPrimaryCollection.Error != nil {
				return resultPrimaryCollection.Error
			}
		}
	case Source:
		relationObject.Id = object.ID
		relationObject.settings = object.RelationInfo()
		if query != nil {
			resultPrimaryCollection := query.First(&object)
			if resultPrimaryCollection.Error != nil {
				return resultPrimaryCollection.Error
			}
		}
	default:
		return fmt.Errorf("can't check presence of primary resource")
	}

	return nil
}

func NewRelationObject(objectModel interface{}, currentTenantID int64, db *gorm.DB) (RelationObject, error) {
	object := RelationObject{baseObject: objectModel, CurrentTenantID: currentTenantID}
	err := object.setRelationInfo(db)
	return object, err
}
