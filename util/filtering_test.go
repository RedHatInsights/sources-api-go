package util

import "testing"

var columnNameTests = []struct {
	name     string
	input    string
	expected bool
}{
	{"simple lowercase", "name", true},
	{"simple id", "id", true},
	{"underscore separated", "created_at", true},
	{"compound name", "source_type_id", true},
	{"leading underscore", "_private", true},
	{"single char", "a", true},
	{"mixed case", "SourceType", true},
	{"with digits", "field2", true},
	{"all caps", "ID", true},

	{"empty string", "", false},
	{"starts with digit", "1name", false},
	{"contains semicolon", "name;", false},
	{"contains space", "name foo", false},
	{"contains quote", "name'", false},
	{"contains dash", "source-type", false},
	{"contains dot", "table.column", false},
	{"contains paren", "name()", false},
	{"contains star", "*", false},
	{"contains equals", "name=1", false},
	{"sql injection drop", "name;DROP TABLE sources--", false},
	{"sql injection or", "name OR 1=1", false},
	{"sql injection union", "name UNION SELECT", false},
	{"sql injection quote", "name' OR '1'='1", false},
	{"sql injection comment", "name--", false},
	{"percent sign", "name%", false},
}

func TestIsValidColumnName(t *testing.T) {
	for _, tt := range columnNameTests {
		result := IsValidColumnName(tt.input)
		if result != tt.expected {
			t.Errorf("IsValidColumnName(%q) [%s]: got %t, want %t",
				tt.input, tt.name, result, tt.expected)
		}
	}
}
