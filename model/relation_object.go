package model

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/RedHatInsights/sources-api-go/util"
	pluralize "github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type RelationSetting struct {
	RelationType string
	Through      string
}

type RelationObject struct {
	Id              int64
	CurrentTenantID int64
	baseObject      interface{}
}

func (relationObject *RelationObject) HasManyRelation(query *gorm.DB, model interface{}) *gorm.DB {
	expression := []clause.Expression{clause.Eq{
		Column: clause.Column{Table: clause.CurrentTable, Name: relationObject.foreignKeyFrom()},
		Value:  relationObject.Id},
	}

	return query.Clauses(clause.Where{Exprs: expression}).Model(model)
}

func (relationObject *RelationObject) tagsFromRelationDefinedAs(model interface{}) map[string]string {
	elemName := reflect.TypeOf(model).Elem().Name()
	p := pluralize.NewClient()
	relationName := p.Plural(elemName)

	statement := reflect.TypeOf(relationObject.baseObject)
	field, _ := statement.FieldByName(relationName)
	tag := field.Tag.Get("gorm")
	return schema.ParseTagSetting(tag, ";")
}

func (relationObject *RelationObject) HasMany(model interface{}, query *gorm.DB) *gorm.DB {
	for key, throughTable := range relationObject.tagsFromRelationDefinedAs(model) {
		if key == "MANY2MANY" {
			return relationObject.HasManyThrough(query, model, throughTable)
		}
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

func (relationObject *RelationObject) HasManyThrough(query *gorm.DB, model interface{}, throughTable string) *gorm.DB {
	query = query.Debug().Select(relationObject.SelectStatementFor(query, model))
	query.Statement.Distinct = true

	subCollectionModel := strcase.ToSnake(reflect.TypeOf(model).Elem().Name())
	query.Model(model)
	expression := []clause.Expression{clause.Eq{
		Column: clause.Column{Table: clause.CurrentTable, Name: "id"},
		Value:  clause.Column{Table: throughTable, Name: subCollectionModel + "_id"},
	}}

	if relationObject.CurrentTenantID != 0 {
		expression = append(expression, clause.Eq{Column: clause.Column{Table: throughTable, Name: "tenant_id"},
			Value: relationObject.CurrentTenantID})
	}

	joins := append([]clause.Join{}, clause.Join{
		Type:  clause.InnerJoin,
		Table: clause.Table{Name: throughTable},
		ON:    clause.Where{Exprs: expression},
	})

	query.Statement.AddClause(clause.From{Joins: joins})
	query.Where(throughTable+"."+relationObject.foreignKeyFrom()+" = ?", relationObject.Id)

	return query
}

func (relationObject *RelationObject) foreignKeyFrom() string {
	return strcase.ToSnake(reflect.TypeOf(relationObject.baseObject).Name()) + "_id"
}

func (relationObject *RelationObject) setRelationObjectID() error {
	switch object := relationObject.baseObject.(type) {
	case SourceType:
		relationObject.Id = object.Id
	case ApplicationType:
		relationObject.Id = object.Id
	case Source:
		relationObject.Id = object.ID
	default:
		return fmt.Errorf("can't set ID to relation object, object type is not recognized")
	}

	return nil
}

func (relationObject *RelationObject) checkIfPrimaryRecordExists(query *gorm.DB) error {
	result := map[string]interface{}{}

	switch relationObject.baseObject.(type) {
	case Source:
		query.Model(relationObject.baseObject).
			Where("tenant_id = ?", relationObject.CurrentTenantID).
			Find(&result, relationObject.Id)

	case SourceType, ApplicationType:
		query.Model(relationObject.baseObject).
			Find(&result, relationObject.Id)

	default:
		return fmt.Errorf("unexpected primary record type")
	}

	if len(result) == 0 {
		return util.NewErrNotFound(relationObject.StringBaseObject())
	}

	return nil
}

func NewRelationObject(objectModel interface{}, currentTenantID int64, query *gorm.DB) (RelationObject, error) {
	object := RelationObject{baseObject: objectModel, CurrentTenantID: currentTenantID}
	err := object.setRelationObjectID()
	if err != nil {
		return object, err
	}

	err = object.checkIfPrimaryRecordExists(query)
	if err != nil {
		return object, err
	}

	return object, nil
}

func (relationObject *RelationObject) StringBaseObject() string {
	switch relationObject.baseObject.(type) {
	case Application:
		return "application"
	case ApplicationType:
		return "application type"
	case ApplicationAuthentication:
		return "application authentication"
	case Source:
		return "source"
	case SourceType:
		return "source type"
	case MetaData:
		return "metadata"
	case Endpoint:
		return "endpoint"
	default:
		return ""
	}
}
