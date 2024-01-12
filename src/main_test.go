package main

import (
	"testing"
)

func Test_playlistNameFromDate(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{
			"2022-10-02T00:00:00Z",
			"October '22",
		},
		{
			"2023-02-02T00:00:00Z",
			"February '23",
		},
		{
			"2030-09-02T00:00:00Z",
			"September '30",
		},
	}

	for _, c := range cases {
		got := playlistNameFromDateString(c.input)

		if got != c.want {
			t.Errorf("got %s but expected %s", got, c.want)
		}
	}
}
