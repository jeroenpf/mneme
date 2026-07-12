package slug

import "testing"

func TestMake(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Vehicle Listing API", "vehicle-listing-api"},
		{"  Leading/Trailing  ", "leading-trailing"},
		{"!!!", "doc"},
		{"", "doc"},
		{"snake_case Title", "snake-case-title"},
		{"already-kebab", "already-kebab"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"C1-142: Inventory", "c1-142-inventory"},
		{"Mixed-Case_With.Punct!", "mixed-case-with-punct"},
		{"trailing---dashes---", "trailing-dashes"},
	}
	for _, tc := range cases {
		got := Make(tc.in)
		if got != tc.want {
			t.Errorf("Make(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
