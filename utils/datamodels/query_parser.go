package datamodels

import (
	"strings"
)

type ParseResult struct {
	Filters    []Filter
	Sorts      []Sort
	Pagination Pagination
}

func ParseQuery(args StrArgs) ParseResult {
	sorts := []Sort{}
	filters := []Filter{}
	var pagination Pagination

	grouped := groupByFirstIdentifier(args, "[", "]")

	for key, value := range grouped {
		if key == "sort" {
			sorts = parseSorts(value)
		} else if key == "pagination" {
			pagination = parsePagination(value)
		} else {
			filters = append(filters, parseFilter(key, value))
		}
	}

	return ParseResult{
		Filters:    filters,
		Sorts:      sorts,
		Pagination: pagination,
	}
}

func parseFilter(field string, args StrArgs) Filter {
	constraints := []Constraint{}
	matchType := MatchAll

	for key, values := range args {
		if key == "operator" {
			if len(values) > 0 && values[0] == "or" {
				matchType = MatchAny
			}
		} else {
			for _, v := range values {
				val := v
				constraints = append(constraints, Constraint{
					Match:  key,
					Values: []*string{&val},
				})
			}
		}
	}

	return Filter{
		FieldName:   field,
		MatchType:   matchType,
		Constraints: constraints,
	}
}

func parseSorts(args StrArgs) []Sort {
	sorts := []Sort{}
	for key, values := range args {
		if len(values) > 0 {
			order := SortOrderAsc
			if values[0] == "-1" || values[0] == "desc" {
				order = SortOrderDesc
			}
			sorts = append(sorts, Sort{Field: key, Order: order})
		}
	}
	return sorts
}

func parsePagination(args StrArgs) Pagination {
	p := Pagination{}
	if limit, ok := args["limit"]; ok && len(limit) > 0 {
		p.Limit = &limit[0]
	}
	if offset, ok := args["offset"]; ok && len(offset) > 0 {
		p.Offset = &offset[0]
	}
	return p
}

func groupByFirstIdentifier(args StrArgs, start, end string) map[string]StrArgs {
	result := make(map[string]StrArgs)
	for key, value := range args {
		parts := strings.Split(key, start)
		if len(parts) != 2 {
			continue
		}
		k := parts[0]
		subKey := parts[1]
		if !strings.HasSuffix(subKey, end) {
			continue
		}
		subKey = subKey[:len(subKey)-len(end)]

		if _, ok := result[k]; !ok {
			result[k] = make(StrArgs)
		}
		result[k][subKey] = value
	}
	return result
}
