package graph

import (
	"strings"

	generated_model "github.com/RedHatInsights/sources-api-go/graph/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// parses all the arguments for us - both sort_by + filters
func parseArgs(sortBy []*generated_model.SortBy, filters []*generated_model.Filter) []util.Filter {
	return append(parseSortBy(sortBy), parseFilters(filters)...)
}

func parseSortBy(sortBy []*generated_model.SortBy) []util.Filter {
	sorts := make([]util.Filter, len(sortBy))

	// parse the sortBy struct - including using an enum for asc/desc
	for i, sby := range sortBy {
		col := sby.Name
		if trimmed, ok := strings.CutPrefix(col, "source_type."); ok {
			col = trimmed
		}

		dir := "ASC"
		if sby.Direction != nil && sby.Direction.IsValid() && sby.Direction.String() == "desc" {
			dir = "DESC"
		}

		var filter util.Filter
		if _, ok := strings.CutPrefix(sby.Name, "source_type."); ok {
			filter = util.Filter{Operation: "sort_by", Subresource: "source_type", Value: []string{col + " " + dir}}
		} else {
			filter = util.Filter{Operation: "sort_by", Value: []string{col + " " + dir}}
		}

		sorts[i] = filter
	}

	if len(sorts) == 0 {
		sorts = append(sorts, util.Filter{Operation: "sort_by", Value: []string{"id ASC"}})
	}

	return sorts
}

func parseFilters(filters []*generated_model.Filter) []util.Filter {
	outFilters := make([]util.Filter, len(filters))

	// parse the filter struct - including subresource filtering
	for i, f := range filters {
		filter := util.Filter{Value: f.Value}

		// operation can be nil (defaults to ""/eq)
		if f.Operation != nil {
			filter.Operation = *f.Operation
		}

		// handle subresource filtering
		if trimmed, ok := strings.CutPrefix(f.Name, "source_type."); ok {
			filter.Name = trimmed
			filter.Subresource = "source_type"
		} else if trimmed, ok := strings.CutPrefix(f.Name, "applications."); ok {
			filter.Name = trimmed
			filter.Subresource = "application"
		} else {
			filter.Name = f.Name
		}

		outFilters[i] = filter
	}

	return outFilters
}
