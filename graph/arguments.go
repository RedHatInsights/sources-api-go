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
		var filter util.Filter
		if strings.HasPrefix(sby.Name, "source_type.") {
			filter = util.Filter{Operation: "sort_by", Subresource: "source_type", Value: []string{strings.TrimPrefix(sby.Name, "source_type.")}}
		} else {
			filter = util.Filter{Operation: "sort_by", Value: []string{sby.Name}}
		}

		// ascending is default - so we only need to set it to desc if specified
		if sby.Direction != nil && sby.Direction.IsValid() && sby.Direction.String() == "desc" {
			filter.Value = append(filter.Value, "desc")
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
		if strings.HasPrefix(f.Name, "source_type.") {
			filter.Name = strings.TrimPrefix(f.Name, "source_type.")
			filter.Subresource = "source_type"
		} else if strings.HasPrefix(f.Name, "applications.") {
			filter.Name = strings.TrimPrefix(f.Name, "applications.")
			filter.Subresource = "application"
		} else {
			filter.Name = f.Name
		}

		outFilters[i] = filter
	}
	return outFilters
}
