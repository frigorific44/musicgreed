package musicinfo

import (
	"testing"

	"go.uploadedlobster.com/musicbrainzws2"
)

func TestReleaseGroupsByArtist(t *testing.T) {
	client, stop := NewMGClient()
	defer stop()
	mbid := musicbrainzws2.MBID("7e870dd5-2667-454b-9fcf-a132dd8071f1")
	groups, err := ReleaseGroupsByArtist(client, mbid)
	if err != nil {
		t.Fatalf(`ReleaseGroupsByArtist(client, %v) returned error, %q`, mbid, err)
	}
	if len(groups) == 0 {
		t.Fatalf(`ReleaseGroupsByArtist(client, %v) returned empty ReleaseGroups`, mbid)
	}
	for _, rg := range groups {
		if len(rg.Releases) == 0 {
			t.Fatalf(`ReleaseGroupsByArtist(client, %v) returned empty Releases from ReleaseGroup %v`, mbid, rg)
		}
		for _, r := range rg.Releases {
			if len(r.Media) == 0 {
				t.Fatalf(`ReleaseGroupsByArtist(client, %v) returned empty Media from Release %v`, mbid, r)
			}
			for _, m := range r.Media {
				if len(m.Tracks) == 0 {
					t.Fatalf(`ReleaseGroupsByArtist(client, %v) returned empty Tracks from Media %v`, mbid, m)
				}
			}
		}
	}
}
