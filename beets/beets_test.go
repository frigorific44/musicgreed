package beets

import (
	"fmt"
	"strings"
	"testing"
	"time"

	mb2 "go.uploadedlobster.com/musicbrainzws2"
)

func TestArtistTrackTitles(t *testing.T) {
	tracks, err := ArtistTrackTitles(mb2.MBID("8682866a-4f7a-43f5-83b2-06eabd0f2d4c"))
	if err != nil {
		t.Fatalf(`ArtistTrackTitles returned error: %v`, err.Error())
	}
	if len(tracks) < 2 {
		t.Fatalf(`ArtistTrackTitles returned fewer tracks than expected: %+v`, tracks)
	}
	for _, track := range tracks {
		if string(track.ID) == "" {
			t.Errorf(`ArtistTrackTitles returned empty ID on track %+v`, track)
		}
		if track.Title == "" {
			t.Errorf(`ArtistTrackTitles returned empty title on track %+v`, track)
		}
		zeroDuration, _ := time.ParseDuration("0s")
		if track.Length.Duration == zeroDuration {
			t.Errorf(`ArtistTrackTitles returned zero length on track %+v`, track)
		}
		if track.Position == 0 {
			t.Errorf(`ArtistTrackTitles returned zero position on track %+v`, track)
		}
	}
}

func TestUnmarshal(t *testing.T) {
	cases := []struct {
		Want []mb2.Track
	}{
		{
			Want: []mb2.Track{
				{ID: "00000000-0000-0000-0000-000000000000",
					Title:    "A",
					Length:   mb2.Duration{Duration: time.Duration(71 * time.Second)},
					Position: 1},
			},
		},
	}
	badLines := []string{
		"nonsense",
		`{"ID":ca1b4c5d-21bd-45aa-879f-60bbeb10e91e,"Title":"Broken Field","Length":"3:02","Position":}`,
	}
	for _, c := range cases {
		var lines []string
		for _, w := range c.Want {
			lines = append(lines, fmt.Sprintf(trackFormatBase, w.ID, w.Title, w.Length.String(), fmt.Sprintf(`%02d`, w.Position)))
		}
		in := strings.Join(lines, "\n")
		out, err := unmarshalBeetsTracks(in)
		if err != nil {
			t.Errorf(`unmarshalBeetsTracks returned on %v error: %v`, in, err)
		}
		if len(out) != len(c.Want) {
			t.Errorf(`unmarshalBeetsTracks returned %v on "%v", wanted %v`, out, in, c.Want)
			continue
		}
		for i := range c.Want {
			if out[i].ID != c.Want[i].ID || out[i].Title != c.Want[i].Title || out[i].Length.String() != c.Want[i].Length.String() || out[i].Position != c.Want[i].Position {
				t.Errorf(`unmarshalBeetsTracks returned %v on "%v", wanted %v`, out, in, c.Want)
			}
		}
		// Insert each bad case to confirm the unspoiled lines still come out fine.
		for _, bad := range badLines {
			var spoiledCases [][]string
			spoiledCases = append(spoiledCases, append(lines, bad))
			spoiledCases = append(spoiledCases, append(lines[:1], append([]string{bad}, lines[1:]...)...))
			spoiledCases = append(spoiledCases, append([]string{bad}, lines...))
			for _, spoiledCase := range spoiledCases {
				in := strings.Join(spoiledCase, "\n")
				out, err := unmarshalBeetsTracks(in)
				if err == nil {
					t.Error("unmarshalBeetsTracks did not return an error")
				}
				if len(out) != len(lines) {
					t.Errorf(`unmarshalBeetsTracks(%v) = %+v did not return enough unspoiled tracks`, in, out)
				}
			}
		}
	}
}
