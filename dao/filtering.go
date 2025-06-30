package dao

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

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
	)

	for _, filter := range filters {
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
		case "sort_by":
			// prepend the table name if it was set
			filter.Value[0] = filterName + filter.Value[0]
			query = query.Order(strings.Join(filter.Value, " "))
		default:
			return nil, fmt.Errorf("unsupported operation %v", filter.Operation)
		}
	}

	return query, nil
}
