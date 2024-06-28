package cmd

import (
	"fmt"
	"math/rand"
	"slices"
	"strconv"
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
		if res := uniqueReleases(c.Releases, setCoverConfig{}); len(res) != c.ResultLen {
			t.Errorf(`removeDuplicateReleases(%v) = %v, wanted %v result(s)`, c.Releases, res, c.ResultLen)
		}
	}
}

func TestCoverCombinations(t *testing.T) {
	cases := []struct {
		TrackMap map[string][]int
		Want     [][]int
	}{
		{TrackMap: map[string][]int{
			"a": {0},
			"b": {1},
		}, Want: [][]int{{0, 1}}},
		{TrackMap: map[string][]int{
			"a": {0, 1},
			"b": {1, 0},
		}, Want: [][]int{{0}, {1}}},
		{TrackMap: map[string][]int{
			"a": {0},
			"b": {1, 0},
			"c": {1, 2},
		}, Want: [][]int{{0, 1}, {0, 2}}},
	}

	for _, c := range cases {
		res := coverCombinations(c.TrackMap)
		if len(res) != len(c.Want) {
			t.Errorf(`coverCombinations(%v) = %v, wanted %v`, c.TrackMap, res, c.Want)
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
			if !resMap[s] {
				t.Errorf(`coverCombinations(%v) = %v, wanted %v`, c.TrackMap, res, c.Want)
				break
			}
		}
	}
}

func genTrackMap(t int, r int) map[string][]int {
	random := rand.New(rand.NewSource(99))
	trackMap := make(map[string][]int)
	for i := 0; i < t; i++ {
		releases := random.Perm(r)[:random.Intn(r)]
		trackMap[strconv.Itoa(i)] = releases
	}
	return trackMap
}

var cover_combinations_table = []struct {
	tracks   int
	releases int
}{
	{25, 10},
	{50, 15},
	{100, 20},
}

func BenchmarkCoverCombinations(b *testing.B) {
	for _, v := range cover_combinations_table {
		b.Run(fmt.Sprintf("%+v", v), func(b *testing.B) {
			trackMap := genTrackMap(v.tracks, v.releases)
			for i := 0; i < b.N; i++ {
				coverCombinations(trackMap)
			}
		})
	}
}
