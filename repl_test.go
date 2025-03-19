package main

import "testing"

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "  hello  world  ",
			expected: []string{"hello", "world"},
		},
		{
			input:    "Charmander Bulbasaur PIKACHU",
			expected: []string{"charmander", "bulbasaur", "pikachu"},
		},
	}

	for _, c := range cases {
		actual := cleanInput(c.input)
		if len(actual) != len(c.expected) {
			t.Errorf("For input '%s', expected length %d but got %d", c.input, len(c.expected), len(actual))
		}

		for i := range actual {
			if actual[i] != c.expected[i] {
				t.Errorf("For input '%s', expected '%s' but got '%s' at index %d", c.input, c.expected[i], actual[i], i)
			}
		}
	}
}
