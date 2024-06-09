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

func TestCoverPermutationsRecursive(t *testing.T) {
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
		res := coverPermutationsRecursive(c.TrackMap, c.Curr)
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
