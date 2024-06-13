package cmd

import (
	"fmt"
	"slices"
	"testing"

	mb2 "go.uploadedlobster.com/musicbrainzws2"
)

func TestRemoveDuplicateReleases(t *testing.T) {
	r := []mb2.Release{
		{Media: []mb2.Medium{{Tracks: []mb2.Track{
			{Title: "a"},
			{Title: "b"},
			{Title: "c"},
		}}}},
		{Media: []mb2.Medium{{Tracks: []mb2.Track{
			{Title: "b"},
			{Title: "c"},
			{Title: "a"},
		}}}},
		{Media: []mb2.Medium{{Tracks: []mb2.Track{
			{Title: "d"},
			{Title: "e"},
			{Title: "f"},
		}}}},
		{Media: []mb2.Medium{{Tracks: []mb2.Track{
			{Title: "b"},
			{Title: "c"},
			{Title: "d"},
		}}}},
	}
	cases := []struct {
		Releases  []mb2.Release
		ResultLen int
	}{
		{Releases: []mb2.Release{r[0], r[0]}, ResultLen: 1},
		{Releases: []mb2.Release{r[0], r[1]}, ResultLen: 1},
		{Releases: []mb2.Release{r[0], r[2]}, ResultLen: 2},
		{Releases: []mb2.Release{r[0], r[3]}, ResultLen: 2},
		{Releases: []mb2.Release{r[1], r[2]}, ResultLen: 2},
		{Releases: []mb2.Release{r[1], r[3]}, ResultLen: 2},
		{Releases: []mb2.Release{r[2], r[3]}, ResultLen: 2},
		{Releases: []mb2.Release{r[0], r[1], r[2]}, ResultLen: 2},
		{Releases: []mb2.Release{r[0], r[1], r[3]}, ResultLen: 2},
		{Releases: []mb2.Release{r[0], r[2], r[3]}, ResultLen: 3},
	}
	for _, c := range cases {
		if res := removeDuplicateReleases(c.Releases); len(res) != c.ResultLen {
			t.Errorf(`removeDuplicateReleases(%v) = %v, wanted %v result(s)`, c.Releases, res, c.ResultLen)
		}
	}
}

func TestCoverPermutations(t *testing.T) {
	cases := []struct {
		TrackMap map[string][]int
		Curr     []int
		Want     [][]int
	}{
		{TrackMap: map[string][]int{
			"a": {0},
			"b": {1},
		}, Curr: []int{}, Want: [][]int{{0, 1}}},
		{TrackMap: map[string][]int{
			"a": {0, 1},
			"b": {1, 0},
		}, Curr: []int{}, Want: [][]int{{0}, {1}}},
		{TrackMap: map[string][]int{
			"a": {0},
			"b": {1, 0},
			"c": {1, 2},
		}, Curr: []int{}, Want: [][]int{{0, 1}, {0, 2}}},
	}

	for _, c := range cases {
		res := coverPermutations(c.TrackMap, c.Curr)
		if len(res) != len(c.Want) {
			t.Errorf(`coverPermutationsRecursive(%v, %v) = %v, wanted %v`, c.TrackMap, c.Curr, res, c.Want)
			continue
		}
		resMap := make(map[string]bool)
		for _, cover := range res {
			slices.Sort(cover)
			resMap[fmt.Sprint(cover)] = true
		}
		for _, wCover := range c.Want {
			slices.Sort(wCover)
			s := fmt.Sprint(wCover)
			_, ok := resMap[s]
			if !ok {
				t.Errorf(`coverPermutationsRecursive(%v, %v) = %v, wanted %v`, c.TrackMap, c.Curr, res, c.Want)
				break
			}
		}
	}
}

func BenchmarkCoverPermutations(b *testing.B) {
	trackMap := map[string][]int{
		"a": {1, 8, 4, 9, 3},
		"b": {0, 9, 4, 7, 6},
		"c": {9, 5, 0, 1, 2},
		"d": {9, 3, 7, 6, 1},
		"e": {4, 0, 6, 3, 2},
		"f": {2, 4, 3, 5, 1},
		"g": {7, 4, 2, 6, 9},
		"h": {9, 1, 2, 5, 0},
		"i": {7, 0, 5, 9, 3},
		"j": {7, 1, 5, 9, 0},
		"k": {5, 3, 0, 4, 9},
		"l": {1, 3, 7, 5, 6},
		"m": {4, 3, 6, 8, 7},
		"n": {4, 2, 1, 9, 0},
		"o": {9, 8, 4, 3, 5},
		"p": {8, 9, 6, 4, 0},
		"q": {9, 3, 7, 1, 2},
		"r": {3, 8, 1, 0, 7},
		"s": {5, 9, 1, 6, 2},
		"t": {7, 0, 5, 4, 6},
		"u": {7, 5, 3, 1, 6},
		"v": {9, 4, 5, 2, 1},
		"w": {3, 2, 1, 0, 5},
		"x": {8, 4, 1, 9, 2},
		"y": {3, 7, 1, 8, 4},
		"z": {3, 8, 5, 2, 7},
	}
	for i := 0; i < b.N; i++ {
		coverPermutations(trackMap, []int{})
	}
}
