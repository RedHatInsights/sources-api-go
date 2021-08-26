package dao

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/middleware"
	"gorm.io/gorm"
)

func applyFilters(query *gorm.DB, filters []middleware.Filter) error {
	if query.Statement.Table == "" {
		query.Statement.Parse(query.Statement.Model)
	}
	var filterName string
	for _, filter := range filters {
		if query.Statement.Table != "" {
			filterName = fmt.Sprintf("%v.%v", query.Statement.Table, filter.Name)
		} else {
			filterName = filter.Name
		}

		switch filter.Operation {
		case "", "[eq]":
			query = query.Where(fmt.Sprintf("%v = ?", filterName), filter.Value[0])
		case "[not_eq]":
			query = query.Where(fmt.Sprintf("%v != ?", filterName), filter.Value[0])
		case "[gt]":
			query = query.Where(fmt.Sprintf("%v > ?", filterName), filter.Value[0])
		case "[gte]":
			query = query.Where(fmt.Sprintf("%v >= ?", filterName), filter.Value[0])
		case "[lt]":
			query = query.Where(fmt.Sprintf("%v < ?", filterName), filter.Value[0])
		case "[lte]":
			query = query.Where(fmt.Sprintf("%v <= ?", filterName), filter.Value[0])
		case "[nil]":
			query = query.Where(fmt.Sprintf("%v IS NULL", filterName))
		case "[not_nil]":
			query = query.Where(fmt.Sprintf("%v IS NOT NULL", filterName))
		case "[contains]":
			query = query.Where(fmt.Sprintf("%v LIKE ?", filterName), fmt.Sprintf("%%%s%%", filter.Value[0]))
		case "[starts_with]":
			query = query.Where(fmt.Sprintf("%v LIKE ?", filterName), fmt.Sprintf("%s%%", filter.Value[0]))
		case "[ends_with]":
			query = query.Where(fmt.Sprintf("%v LIKE ?", filterName), fmt.Sprintf("%%%s", filter.Value[0]))
		case "[eq_i]":
			query = query.Where(fmt.Sprintf("LOWER(%v) = ?", filterName), strings.ToLower(filter.Value[0]))
		case "[not_eq_i]":
			query = query.Where(fmt.Sprintf("LOWER(%v) != ?", filterName), strings.ToLower(filter.Value[0]))
		case "[contains_i]":
			query = query.Where(fmt.Sprintf("%v ILIKE ?", filterName), fmt.Sprintf("%%%s%%", filter.Value[0]))
		case "[starts_with_i]":
			query = query.Where(fmt.Sprintf("%v ILIKE ?", filterName), fmt.Sprintf("%s%%", filter.Value[0]))
		case "[ends_with_i]":
			query = query.Where(fmt.Sprintf("%v ILIKE ?", filterName), fmt.Sprintf("%%%s", filter.Value[0]))
		default:
			return fmt.Errorf("unsupported operation %v", filter.Operation)
		}
	}

	return nil
}
