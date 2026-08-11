package graph

import (
	"testing"

	generated_model "github.com/RedHatInsights/sources-api-go/graph/model"
)

func TestParseSortByDefaultAsc(t *testing.T) {
	dir := generated_model.DirectionAsc
	sorts := parseSortBy([]*generated_model.SortBy{
		{Name: "name", Direction: &dir},
	})

	if len(sorts) != 1 {
		t.Fatalf("expected 1 sort, got %d", len(sorts))
	}

	if len(sorts[0].Value) != 1 {
		t.Fatalf("expected 1 value element, got %d", len(sorts[0].Value))
	}

	if sorts[0].Value[0] != "name ASC" {
		t.Errorf("expected %q, got %q", "name ASC", sorts[0].Value[0])
	}
}

func TestParseSortByDesc(t *testing.T) {
	dir := generated_model.DirectionDesc
	sorts := parseSortBy([]*generated_model.SortBy{
		{Name: "name", Direction: &dir},
	})

	if len(sorts) != 1 {
		t.Fatalf("expected 1 sort, got %d", len(sorts))
	}

	if sorts[0].Value[0] != "name DESC" {
		t.Errorf("expected %q, got %q", "name DESC", sorts[0].Value[0])
	}
}

func TestParseSortByNilDirection(t *testing.T) {
	sorts := parseSortBy([]*generated_model.SortBy{
		{Name: "id"},
	})

	if sorts[0].Value[0] != "id ASC" {
		t.Errorf("expected %q, got %q", "id ASC", sorts[0].Value[0])
	}
}

func TestParseSortByEmpty(t *testing.T) {
	sorts := parseSortBy(nil)

	if len(sorts) != 1 {
		t.Fatalf("expected default sort, got %d sorts", len(sorts))
	}

	if sorts[0].Value[0] != "id ASC" {
		t.Errorf("expected default %q, got %q", "id ASC", sorts[0].Value[0])
	}
}

func TestParseSortBySourceTypeSubresource(t *testing.T) {
	dir := generated_model.DirectionDesc
	sorts := parseSortBy([]*generated_model.SortBy{
		{Name: "source_type.vendor", Direction: &dir},
	})

	if sorts[0].Subresource != "source_type" {
		t.Errorf("expected subresource %q, got %q", "source_type", sorts[0].Subresource)
	}

	if sorts[0].Value[0] != "vendor DESC" {
		t.Errorf("expected %q, got %q", "vendor DESC", sorts[0].Value[0])
	}
}

func TestParseSortByProducesSingleValueElement(t *testing.T) {
	dir := generated_model.DirectionDesc
	sorts := parseSortBy([]*generated_model.SortBy{
		{Name: "name", Direction: &dir},
	})

	if len(sorts[0].Value) != 1 {
		t.Errorf("expected col and dir combined into 1 value element, got %d: %v", len(sorts[0].Value), sorts[0].Value)
	}
}

func TestParseSortByMultiple(t *testing.T) {
	asc := generated_model.DirectionAsc
	desc := generated_model.DirectionDesc
	sorts := parseSortBy([]*generated_model.SortBy{
		{Name: "name", Direction: &asc},
		{Name: "id", Direction: &desc},
	})

	if len(sorts) != 2 {
		t.Fatalf("expected 2 sorts, got %d", len(sorts))
	}

	if sorts[0].Value[0] != "name ASC" {
		t.Errorf("expected %q, got %q", "name ASC", sorts[0].Value[0])
	}

	if sorts[1].Value[0] != "id DESC" {
		t.Errorf("expected %q, got %q", "id DESC", sorts[1].Value[0])
	}
}
