package musicinfo

import (
	"testing"

	"go.uploadedlobster.com/musicbrainzws2"
)

func TestBuildArtist(t *testing.T) {
	client, stop := NewMGClient()
	defer stop()
	mbid := musicbrainzws2.MBID("482ee758-1245-42e2-8e4c-4978e69193f4")
	artist, err := BuildArtist(client, mbid)
	if err != nil {
		t.Fatalf(`BuildArtist(client, %v) returned error, %q`, mbid, err)
	}
	if len(artist.ReleaseGroups) == 0 {
		t.Fatalf(`BuildArtist(client, %v) returned empty ReleaseGroups`, mbid)
	}
	for _, rg := range artist.ReleaseGroups {
		if len(rg.Releases) == 0 {
			t.Fatalf(`BuildArtist(client, %v) returned empty Releases from ReleaseGroup %v`, mbid, rg)
		}
		for _, r := range rg.Releases {
			if len(r.Media) == 0 {
				t.Fatalf(`BuildArtist(client, %v) returned empty Media from Release %v`, mbid, r)
			}
			for _, m := range r.Media {
				if len(m.Tracks) == 0 {
					t.Fatalf(`BuildArtist(client, %v) returned empty Tracks from Media %v`, mbid, m)
				}
			}
		}
	}
}
