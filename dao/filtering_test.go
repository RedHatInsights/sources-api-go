package dao

import (
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

// --- isColumnAllowed ---

var isColumnAllowedTests = []struct {
	name        string
	table       string
	subresource string
	column      string
	expected    bool
}{
	// valid table + valid column
	{"sources id", "sources", "", "id", true},
	{"sources name", "sources", "", "name", true},
	{"sources created_at", "sources", "", "created_at", true},
	{"sources source_type_id", "sources", "", "source_type_id", true},
	{"applications source_id", "applications", "", "source_id", true},
	{"endpoints host", "endpoints", "", "host", true},
	{"rhc_connections rhc_id", "rhc_connections", "", "rhc_id", true},
	{"meta_data step", "meta_data", "", "step", true},
	{"source_types vendor", "source_types", "", "vendor", true},
	{"application_types display_name", "application_types", "", "display_name", true},
	{"authentications authtype", "authentications", "", "authtype", true},
	{"application_authentications application_id", "application_authentications", "", "application_id", true},

	// valid table + column NOT in allowlist
	{"sources tenant_id rejected", "sources", "", "tenant_id", false},
	{"sources password rejected", "sources", "", "password", false},
	{"sources nonexistent rejected", "sources", "", "nonexistent_column", false},
	{"applications tenant_id rejected", "applications", "", "tenant_id", false},
	{"authentications password rejected", "authentications", "", "password", false},
	{"empty column", "sources", "", "", false},

	// unknown table
	{"unknown table", "nonexistent_table", "", "id", false},
	{"tenants table not allowed", "tenants", "", "id", false},

	// empty table
	{"empty table", "", "", "id", false},

	// subresource lookup -> resolves to correct table
	{"subresource source_type name", "sources", "source_type", "name", true},
	{"subresource source_type vendor", "sources", "source_type", "vendor", true},
	{"subresource application_type name", "applications", "application_type", "name", true},
	{"subresource application_type display_name", "applications", "application_type", "display_name", true},
	{"subresource application source_id", "sources", "application", "source_id", true},

	// subresource with column NOT in subresource table's allowlist
	{"subresource source_type tenant_id", "sources", "source_type", "tenant_id", false},
	{"subresource source_type password", "sources", "source_type", "password", false},
	{"subresource application_type bogus", "applications", "application_type", "bogus", false},

	// unknown subresource falls through to base table
	{"unknown subresource falls through", "sources", "unknown_sub", "name", true},
	{"unknown subresource column not in base", "sources", "unknown_sub", "tenant_id", false},
}

func TestIsColumnAllowed(t *testing.T) {
	for _, tt := range isColumnAllowedTests {
		result := isColumnAllowed(tt.table, tt.subresource, tt.column)
		if result != tt.expected {
			t.Errorf("isColumnAllowed(%q, %q, %q) [%s]: got %t, want %t",
				tt.table, tt.subresource, tt.column, tt.name, result, tt.expected)
		}
	}
}

// --- subresource maps ---

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

func TestSubresourceToAliasUsesQuotedIdentifiers(t *testing.T) {
	for sub, alias := range subresourceToAlias {
		if !strings.HasPrefix(alias, `"`) || !strings.HasSuffix(alias, `"`) {
			t.Errorf("subresourceToAlias[%q] = %q, expected quoted identifier", sub, alias)
		}
	}
}

// --- applySortBy ---

func newDryRunQuery(table string) *gorm.DB {
	db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})
	query := db.Session(&gorm.Session{DryRun: true})
	query.Statement = &gorm.Statement{DB: db}
	query.Statement.Table = table

	return query
}

func TestApplySortByValidColumnAsc(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name ASC"}}

	result, err := applySortBy(query, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil query")
	}
}

func TestApplySortByValidColumnDesc(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name DESC"}}

	_, err := applySortBy(query, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplySortByValidColumnLowercaseDir(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name asc"}}

	_, err := applySortBy(query, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplySortByDefaultsToAsc(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name"}}

	_, err := applySortBy(query, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplySortByMultipleValues(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name ASC", "id DESC"}}

	_, err := applySortBy(query, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplySortByInvalidColumnName(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name;DROP TABLE sources-- ASC"}}

	_, err := applySortBy(query, filter)
	if err == nil {
		t.Fatal("expected error for invalid column name, got nil")
	}
}

func TestApplySortByColumnNotInAllowlist(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"tenant_id ASC"}}

	_, err := applySortBy(query, filter)
	if err == nil {
		t.Fatal("expected error for column not in allowlist, got nil")
	}
}

func TestApplySortByInvalidDirection(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name SIDEWAYS"}}

	_, err := applySortBy(query, filter)
	if err == nil {
		t.Fatal("expected error for invalid direction, got nil")
	}
}

func TestApplySortByTooManyParts(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name ASC extra"}}

	_, err := applySortBy(query, filter)
	if err == nil {
		t.Fatal("expected error for too many parts, got nil")
	}
}

func TestApplySortByEmptyValue(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{""}}

	_, err := applySortBy(query, filter)
	if err == nil {
		t.Fatal("expected error for empty value, got nil")
	}
}

func TestApplySortBySqlInjectionInColumn(t *testing.T) {
	injections := []string{
		"(SELECT pg_sleep(5)) ASC",
		"name; DELETE FROM sources-- ASC",
		"CASE WHEN 1=1 THEN name ELSE id END",
		"name ASC; DROP TABLE sources--",
	}

	for _, injection := range injections {
		query := newDryRunQuery("sources")
		filter := util.Filter{Operation: "sort_by", Value: []string{injection}}

		_, err := applySortBy(query, filter)
		if err == nil {
			t.Errorf("expected error for sort_by injection %q, got nil", injection)
		}
	}
}

func TestApplySortBySqlInjectionInDirection(t *testing.T) {
	query := newDryRunQuery("sources")
	filter := util.Filter{Operation: "sort_by", Value: []string{"name ASC,1=1"}}

	_, err := applySortBy(query, filter)
	if err == nil {
		t.Fatal("expected error for injection in direction, got nil")
	}
}

func TestApplySortByUnknownTable(t *testing.T) {
	query := newDryRunQuery("nonexistent_table")
	filter := util.Filter{Operation: "sort_by", Value: []string{"id ASC"}}

	_, err := applySortBy(query, filter)
	if err == nil {
		t.Fatal("expected error for unknown table, got nil")
	}
}

// --- applyFilters validation ---

func TestApplyFiltersRejectsInvalidColumnName(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Name: "name;DROP TABLE", Value: []string{"test"}},
	}

	_, err := applyFilters(query, filters)
	if err == nil {
		t.Fatal("expected error for invalid column name, got nil")
	}
}

func TestApplyFiltersRejectsColumnNotInAllowlist(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Name: "tenant_id", Value: []string{"1"}},
	}

	_, err := applyFilters(query, filters)
	if err == nil {
		t.Fatal("expected error for column not in allowlist, got nil")
	}
}

func TestApplyFiltersAcceptsValidColumn(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Name: "name", Value: []string{"test"}},
	}

	_, err := applyFilters(query, filters)
	if err != nil {
		t.Fatalf("unexpected error for valid column: %v", err)
	}
}

func TestApplyFiltersAcceptsValidSubresourceColumn(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Subresource: "source_type", Name: "name", Value: []string{"amazon"}},
	}

	_, err := applyFilters(query, filters)
	if err != nil {
		t.Fatalf("unexpected error for valid subresource column: %v", err)
	}
}

func TestApplyFiltersRejectsInvalidSubresourceColumn(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Subresource: "source_type", Name: "tenant_id", Value: []string{"1"}},
	}

	_, err := applyFilters(query, filters)
	if err == nil {
		t.Fatal("expected error for invalid subresource column, got nil")
	}
}

func TestApplyFiltersDelegatesToApplySortBy(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Operation: "sort_by", Value: []string{"name ASC"}},
	}

	_, err := applyFilters(query, filters)
	if err != nil {
		t.Fatalf("unexpected error for sort_by delegation: %v", err)
	}
}

func TestApplyFiltersSortByRejectsInjection(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Operation: "sort_by", Value: []string{"name; DROP TABLE sources-- ASC"}},
	}

	_, err := applyFilters(query, filters)
	if err == nil {
		t.Fatal("expected error for sort_by injection via applyFilters, got nil")
	}
}

func TestApplyFiltersRejectsEmptyValue(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Name: "name", Value: []string{}},
	}

	_, err := applyFilters(query, filters)
	if err == nil {
		t.Fatal("expected error for empty filter value, got nil")
	}
}

func TestApplyFiltersMultipleFilters(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Name: "name", Value: []string{"test"}},
		{Name: "id", Value: []string{"1"}},
		{Operation: "sort_by", Value: []string{"name ASC"}},
	}

	_, err := applyFilters(query, filters)
	if err != nil {
		t.Fatalf("unexpected error for multiple valid filters: %v", err)
	}
}

func TestApplyFiltersRejectsOnFirstBadFilter(t *testing.T) {
	query := newDryRunQuery("sources")
	filters := []util.Filter{
		{Name: "name", Value: []string{"test"}},
		{Name: "tenant_id", Value: []string{"1"}},
		{Name: "id", Value: []string{"2"}},
	}

	_, err := applyFilters(query, filters)
	if err == nil {
		t.Fatal("expected error when one filter has disallowed column, got nil")
	}
}

func TestApplyFiltersSqlInjectionInFilterName(t *testing.T) {
	injections := []string{
		"name OR 1=1--",
		"name; DROP TABLE sources",
		"name' UNION SELECT * FROM tenants--",
		"1 OR 1=1",
	}

	for _, injection := range injections {
		query := newDryRunQuery("sources")
		filters := []util.Filter{
			{Name: injection, Value: []string{"test"}},
		}

		_, err := applyFilters(query, filters)
		if err == nil {
			t.Errorf("expected error for filter name injection %q, got nil", injection)
		}
	}
}

// --- allowedFilterColumns coverage ---

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

func TestAllowedFilterColumnsCoversAllTables(t *testing.T) {
	expectedTables := []string{
		"sources", "applications", "application_types",
		"application_authentications", "authentications",
		"endpoints", "meta_data", "rhc_connections",
		"source_rhc_connections", "source_types",
	}

	for _, table := range expectedTables {
		if _, ok := allowedFilterColumns[table]; !ok {
			t.Errorf("allowedFilterColumns missing table %q", table)
		}
	}
}

func TestAllowedFilterColumnsSensitiveColumnsExcluded(t *testing.T) {
	sensitiveColumns := []struct {
		table  string
		column string
	}{
		{"sources", "tenant_id"},
		{"applications", "tenant_id"},
		{"endpoints", "tenant_id"},
		{"authentications", "password"},
		{"authentications", "extra"},
		{"authentications", "tenant_id"},
	}

	for _, sc := range sensitiveColumns {
		if isColumnAllowed(sc.table, "", sc.column) {
			t.Errorf("sensitive column %s.%s should not be in allowlist", sc.table, sc.column)
		}
	}
}

// --- IsValidColumnName ---

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
