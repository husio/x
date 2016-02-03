package webhooks

import "testing"

func TestFindTagRx(t *testing.T) {
	var testCases = []struct {
		ok    bool
		input string
	}{
		{true, "[votehub]"},
		{true, "[votehub foobar]"},
		{false, "[votehub too long xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx]"},
		{false, "[votehub ]"},
		{false, "[votehub    ]"},
	}

	for i, tc := range testCases {
		if findTagsRx.MatchString(tc.input) != tc.ok {
			t.Errorf("%d: want %v: %s", i, tc.ok, tc.input)
		}
	}
}
