package dao

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

var allowedFilterColumns = map[string]map[string]bool{
	"sources": {
		"id": true, "created_at": true, "updated_at": true, "paused_at": true,
		"name": true, "uid": true, "version": true, "imported": true,
		"source_ref": true, "app_creation_workflow": true, "source_type_id": true,
		"availability_status": true, "last_checked_at": true, "last_available_at": true,
	},
	"applications": {
		"id": true, "created_at": true, "updated_at": true, "paused_at": true,
		"availability_status": true, "last_checked_at": true, "last_available_at": true,
		"availability_status_error": true, "extra": true,
		"source_id": true, "application_type_id": true,
	},
	"application_types": {
		"id": true, "created_at": true, "updated_at": true,
		"name": true, "display_name": true, "dependent_applications": true,
		"supported_source_types": true, "supported_authentication_types": true,
		"resource_ownership": true,
	},
	"application_authentications": {
		"id": true, "created_at": true, "updated_at": true, "paused_at": true,
		"application_id": true, "authentication_id": true,
	},
	"authentications": {
		"id": true, "name": true, "authtype": true, "username": true,
		"availability_status": true, "last_checked_at": true, "last_available_at": true,
		"availability_status_error": true,
		"source_id":                 true, "resource_type": true, "resource_id": true,
	},
	"endpoints": {
		"id": true, "created_at": true, "updated_at": true, "paused_at": true,
		"role": true, "port": true, "default": true, "scheme": true,
		"host": true, "path": true, "verify_ssl": true,
		"certificate_authority": true, "receptor_node": true,
		"availability_status": true, "last_checked_at": true, "last_available_at": true,
		"availability_status_error": true, "source_id": true,
	},
	"meta_data": {
		"id": true, "created_at": true, "updated_at": true,
		"step": true, "name": true, "payload": true,
		"substitutions": true, "type": true, "application_type_id": true,
	},
	"rhc_connections": {
		"id": true, "rhc_id": true, "extra": true,
		"availability_status": true, "last_checked_at": true, "last_available_at": true,
		"availability_status_error": true, "created_at": true, "updated_at": true,
	},
	"source_rhc_connections": {
		"source_id": true, "rhc_connection_id": true,
	},
	"source_types": {
		"id": true, "created_at": true, "updated_at": true,
		"category": true, "name": true, "product_name": true,
		"vendor": true, "icon_url": true, "schema": true,
	},
}

var subresourceToTable = map[string]string{
	"source_type":      "source_types",
	"application_type": "application_types",
	"application":      "applications",
}

var subresourceToAlias = map[string]string{
	"source_type":      `"SourceType"`,
	"application_type": `"ApplicationType"`,
	"application":      `"Applications"`,
}

func isColumnAllowed(table, subresource, column string) bool {
	lookupTable := table

	if subresource != "" {
		if t, ok := subresourceToTable[subresource]; ok {
			lookupTable = t
		}
	}

	allowed, ok := allowedFilterColumns[lookupTable]
	if !ok {
		return false
	}

	return allowed[column]
}

func applyFilters(query *gorm.DB, filters []util.Filter) (*gorm.DB, error) {
	if query.Statement.Table == "" {
		err := query.Statement.Parse(query.Statement.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to parse statement: %v", err)
		}
	}

	var (
		filterName    string
		alreadyJoined = make(map[string]bool)
		needsDistinct bool
	)

	for _, filter := range filters {
		if filter.Operation == "sort_by" {
			if filter.Subresource != "" {
				switch filter.Subresource {
				case "source_type":
					if query.Statement.Table != "sources" {
						return nil, fmt.Errorf("cannot sort by source_type subresource for table %q", query.Statement.Table)
					}

					if !alreadyJoined[filter.Subresource] {
						query = query.Joins("SourceType")
						alreadyJoined[filter.Subresource] = true
					}
				case "application_type":
					if query.Statement.Table != "applications" {
						return nil, fmt.Errorf("cannot sort by application_type subresource for table %q", query.Statement.Table)
					}

					if !alreadyJoined[filter.Subresource] {
						query = query.Joins("ApplicationType")
						alreadyJoined[filter.Subresource] = true
					}
				case "application":
					if query.Statement.Table != "sources" {
						return nil, fmt.Errorf("cannot sort by applications subresource for table %q", query.Statement.Table)
					}

					if !alreadyJoined[filter.Subresource] {
						query = query.Joins("Applications")
						alreadyJoined[filter.Subresource] = true
						needsDistinct = true
					}
				default:
					return nil, fmt.Errorf("invalid subresource type [%v]", filter.Subresource)
				}
			}

			var err error
			query, err = applySortBy(query, filter)
			if err != nil {
				return nil, err
			}

			continue
		}

		if filter.Name != "" && !util.IsValidColumnName(filter.Name) {
			return nil, fmt.Errorf("invalid filter parameter")
		}

		if filter.Name != "" && !isColumnAllowed(query.Statement.Table, filter.Subresource, filter.Name) {
			return nil, fmt.Errorf("invalid filter parameter")
		}

		// subresource filtering!
		if filter.Subresource != "" {
			switch filter.Subresource {
			case "source_type":
				if query.Statement.Table != "sources" {
					return nil, fmt.Errorf("cannot filter based on source_type subresource for table %q", query.Statement.Table)
				}

				if !alreadyJoined[filter.Subresource] {
					query = query.Joins("SourceType")
					alreadyJoined[filter.Subresource] = true
				}

				filterName = fmt.Sprintf("%v.%v", `"SourceType"`, filter.Name)
			case "application_type":
				if query.Statement.Table != "applications" {
					return nil, fmt.Errorf("cannot filter based on application_type subresource for table %q", query.Statement.Table)
				}

				if !alreadyJoined[filter.Subresource] {
					query = query.Joins("ApplicationType")
					alreadyJoined[filter.Subresource] = true
				}

				filterName = fmt.Sprintf("%v.%v", `"ApplicationType"`, filter.Name)
			case "application":
				if query.Statement.Table != "sources" {
					return nil, fmt.Errorf("cannot filter based on applications subresource for table %q", query.Statement.Table)
				}

				if !alreadyJoined[filter.Subresource] {
					query = query.Joins(`Applications`)
					alreadyJoined[filter.Subresource] = true
					needsDistinct = true
				}

				filterName = fmt.Sprintf("%v.%v", `"Applications"`, filter.Name)
			default:
				return nil, fmt.Errorf("invalid subresource type [%v]", filter.Subresource)
			}
		} else if query.Statement.Table != "" {
			filterName = fmt.Sprintf("%v.%v", query.Statement.Table, filter.Name)
		} else {
			filterName = filter.Name
		}

		// this can happen sometimes via graphql.
		if len(filter.Value) == 0 {
			return nil, fmt.Errorf("bad filter, no value")
		}

		switch filter.Operation {
		case "", "eq":
			if len(filter.Value) > 1 {
				query = query.Where(fmt.Sprintf("%v IN ?", filterName), filter.Value)
				// distinct since IN apparently can return multiple copies.
				query = query.Distinct()
			} else {
				query = query.Where(fmt.Sprintf("%v = ?", filterName), filter.Value[0])
			}
		case "not_eq":
			query = query.Where(fmt.Sprintf("%v != ?", filterName), filter.Value[0])
		case "gt":
			query = query.Where(fmt.Sprintf("%v > ?", filterName), filter.Value[0])
		case "gte":
			query = query.Where(fmt.Sprintf("%v >= ?", filterName), filter.Value[0])
		case "lt":
			query = query.Where(fmt.Sprintf("%v < ?", filterName), filter.Value[0])
		case "lte":
			query = query.Where(fmt.Sprintf("%v <= ?", filterName), filter.Value[0])
		case "nil":
			query = query.Where(fmt.Sprintf("%v IS NULL", filterName))
		case "not_nil":
			query = query.Where(fmt.Sprintf("%v IS NOT NULL", filterName))
		case "contains":
			query = query.Where(fmt.Sprintf("%v LIKE ?", filterName), fmt.Sprintf("%%%s%%", filter.Value[0]))
		case "starts_with":
			query = query.Where(fmt.Sprintf("%v LIKE ?", filterName), fmt.Sprintf("%s%%", filter.Value[0]))
		case "ends_with":
			query = query.Where(fmt.Sprintf("%v LIKE ?", filterName), fmt.Sprintf("%%%s", filter.Value[0]))
		case "eq_i":
			query = query.Where(fmt.Sprintf("LOWER(%v) = ?", filterName), strings.ToLower(filter.Value[0]))
		case "not_eq_i":
			query = query.Where(fmt.Sprintf("LOWER(%v) != ?", filterName), strings.ToLower(filter.Value[0]))
		case "contains_i":
			query = query.Where(fmt.Sprintf("%v ILIKE ?", filterName), fmt.Sprintf("%%%s%%", filter.Value[0]))
		case "starts_with_i":
			query = query.Where(fmt.Sprintf("%v ILIKE ?", filterName), fmt.Sprintf("%s%%", filter.Value[0]))
		case "ends_with_i":
			query = query.Where(fmt.Sprintf("%v ILIKE ?", filterName), fmt.Sprintf("%%%s", filter.Value[0]))
		default:
			return nil, fmt.Errorf("unsupported operation %v", filter.Operation)
		}
	}

	if needsDistinct {
		query = query.Distinct()
	}

	return query, nil
}

func applySortBy(query *gorm.DB, filter util.Filter) (*gorm.DB, error) {
	var orderClauses []string

	for _, v := range filter.Value {
		parts := strings.Fields(v)
		if len(parts) == 0 || len(parts) > 2 {
			return nil, fmt.Errorf("invalid sort_by parameter")
		}

		col := parts[0]
		if !util.IsValidColumnName(col) {
			return nil, fmt.Errorf("invalid sort_by parameter")
		}

		if !isColumnAllowed(query.Statement.Table, filter.Subresource, col) {
			return nil, fmt.Errorf("invalid sort_by parameter")
		}

		dir := "ASC"

		if len(parts) == 2 {
			d := strings.ToUpper(parts[1])
			if d != "ASC" && d != "DESC" {
				return nil, fmt.Errorf("invalid sort_by parameter")
			}

			dir = d
		}

		tablePrefix := query.Statement.Table

		if filter.Subresource != "" {
			if alias, ok := subresourceToAlias[filter.Subresource]; ok {
				tablePrefix = alias
			}
		}

		if tablePrefix != "" {
			orderClauses = append(orderClauses, fmt.Sprintf("%s.%s %s", tablePrefix, col, dir))
		} else {
			orderClauses = append(orderClauses, fmt.Sprintf("%s %s", col, dir))
		}
	}

	query = query.Order(strings.Join(orderClauses, ", "))

	return query, nil
}
