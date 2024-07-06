package musicinfo

import (
	"fmt"
	"testing"

	"go.uploadedlobster.com/musicbrainzws2"
)

func TestAltTrackExp(t *testing.T) {
	cases := []struct {
		Format string
		Want   bool
	}{
		{`%v`, false},
		{`abc (abc%v)`, false},
		{`abc (abc%vdef)`, false},
		{`abc (%vabc)`, false},
		{`abc (%v)`, true},
		{`abc (abc %v)`, true},
		{`abc (%v abc)`, true},
		{`abc (abc %v def)`, true},
		{`abc (%v.)`, true},
		{`abc (abc %v.)`, true},
		{`abc (%v. abc)`, true},
		{`abc (abc %v. def)`, true},
		{`abc - abc%v`, false},
		{`abc - abc%vdef`, false},
		{`abc - %vabc`, false},
		{`abc - %v`, true},
		{`abc - abc %v`, true},
		{`abc - %v abc`, true},
		{`abc - abc %v def`, true},
		{`abc - %v.`, true},
		{`abc - abc %v.`, true},
		{`abc - %v. abc`, true},
		{`abc - abc %v. def`, true},
	}
	for _, term := range AltTrackTerms {
		for _, c := range cases {
			m := fmt.Sprintf(c.Format, term)
			if AltTrackExp.MatchString(m) != c.Want {
				t.Errorf(`AltTrackExp returned %v on "%v", wanted %v`, !c.Want, m, c.Want)
			}
		}
	}
}

func TestAlmostAltExp(t *testing.T) {
	cases := []struct {
		In   string
		Want bool
	}{
		{`abc`, false},
		{`abc (abc)`, true},
		{`abc - abc`, true},
		{`abc-abc`, false},
	}
	for _, c := range cases {
		if AlmostAltExp.MatchString(c.In) != c.Want {
			t.Errorf(`AlmostAltExp returned %v on "%v", wanted %v`, !c.Want, c.In, c.Want)
		}
	}
}

func TestReleaseGroupsByArtistNotEmpty(t *testing.T) {
	client, stop := NewMGClient()
	defer stop()
	mbid := musicbrainzws2.MBID("7e870dd5-2667-454b-9fcf-a132dd8071f1")
	groups, err := ReleaseGroupsByArtist(client, mbid, "")
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

func TestReleaseGroupsByArtistPagination(t *testing.T) {
	client, stop := NewMGClient()
	defer stop()
	mbid := musicbrainzws2.MBID("a1ed5e33-22ff-4e7d-a457-42f4309e135f")
	groups, err := ReleaseGroupsByArtist(client, mbid, "")
	if err != nil {
		t.Fatalf(`ReleaseGroupsByArtist(client, %v) returned error, %q`, mbid, err)
	}
	if len(groups) < 20 {
		t.Errorf(`ReleaseGroupsByArtist(client, %v) returned %v release groups`, mbid, len(groups))
	}
}
