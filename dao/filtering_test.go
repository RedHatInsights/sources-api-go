package dao

import (
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/util"
)

func TestIsColumnAllowed(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		subresource string
		column      string
		want        bool
	}{
		{"valid source column", "sources", "", "name", true},
		{"valid source id", "sources", "", "id", true},
		{"invalid source column", "sources", "", "password", false},
		{"valid source_type subresource", "sources", "source_type", "name", true},
		{"invalid source_type subresource column", "sources", "source_type", "password", false},
		{"valid application column", "applications", "", "source_id", true},
		{"unknown table", "nonexistent", "", "id", false},
		{"empty column", "sources", "", "", false},
		{"sql injection in column", "sources", "", "name; DROP TABLE sources", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isColumnAllowed(tt.table, tt.subresource, tt.column)
			if got != tt.want {
				t.Errorf("isColumnAllowed(%q, %q, %q) = %v, want %v", tt.table, tt.subresource, tt.column, got, tt.want)
			}
		})
	}
}

func TestSubresourceToAlias(t *testing.T) {
	tests := []struct {
		subresource string
		wantAlias   string
		wantOk      bool
	}{
		{"source_type", `"SourceType"`, true},
		{"application_type", `"ApplicationType"`, true},
		{"application", `"Applications"`, true},
		{"nonexistent", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.subresource, func(t *testing.T) {
			alias, ok := subresourceToAlias[tt.subresource]
			if ok != tt.wantOk {
				t.Errorf("subresourceToAlias[%q]: ok = %v, want %v", tt.subresource, ok, tt.wantOk)
			}

			if alias != tt.wantAlias {
				t.Errorf("subresourceToAlias[%q] = %q, want %q", tt.subresource, alias, tt.wantAlias)
			}
		})
	}
}

func TestSubresourceToTableConsistency(t *testing.T) {
	for sub := range subresourceToAlias {
		if _, ok := subresourceToTable[sub]; !ok {
			t.Errorf("subresourceToAlias has key %q but subresourceToTable does not", sub)
		}
	}

	for sub := range subresourceToTable {
		if _, ok := subresourceToAlias[sub]; !ok {
			t.Errorf("subresourceToTable has key %q but subresourceToAlias does not", sub)
		}
	}
}

func TestAllowedFilterColumnsIntegrity(t *testing.T) {
	for table, columns := range allowedFilterColumns {
		if len(columns) == 0 {
			t.Errorf("table %q has empty allowlist", table)
		}

		for col := range columns {
			if !util.IsValidColumnName(col) {
				t.Errorf("table %q has invalid column name %q in allowlist", table, col)
			}
		}
	}
}

func TestAllowedFilterColumnsExpectedTables(t *testing.T) {
	expectedTables := []string{
		"sources", "applications", "application_types", "application_authentications",
		"authentications", "endpoints", "meta_data", "rhc_connections",
		"source_rhc_connections", "source_types",
	}
	for _, table := range expectedTables {
		if _, ok := allowedFilterColumns[table]; !ok {
			t.Errorf("expected table %q in allowedFilterColumns but not found", table)
		}
	}
}

func TestIsValidColumnNameInjections(t *testing.T) {
	injections := []string{
		"name; DROP TABLE sources--",
		"name OR 1=1--",
		"name' OR '1'='1",
		"1=1",
		"",
		"name UNION SELECT",
		"name;",
		"name--",
	}
	for _, payload := range injections {
		if util.IsValidColumnName(payload) {
			t.Errorf("IsValidColumnName(%q) = true, expected false", payload)
		}
	}
}

func TestIsValidColumnNameValid(t *testing.T) {
	valid := []string{
		"name", "id", "created_at", "source_type_id",
		"availability_status", "last_checked_at", "Name", "ID",
	}
	for _, col := range valid {
		if !util.IsValidColumnName(col) {
			t.Errorf("IsValidColumnName(%q) = false, expected true", col)
		}
	}
}

func TestSubresourceToAliasUsesQuotedIdentifiers(t *testing.T) {
	for sub, alias := range subresourceToAlias {
		if !strings.HasPrefix(alias, `"`) || !strings.HasSuffix(alias, `"`) {
			t.Errorf("subresourceToAlias[%q] = %q, expected quoted identifier", sub, alias)
		}
	}
}
