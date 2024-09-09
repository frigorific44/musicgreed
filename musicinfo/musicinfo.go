package musicinfo

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	mb2 "go.uploadedlobster.com/musicbrainzws2"
)

var (
	AltTrackTermGroups map[string][]string = map[string][]string{
		"Latin":    {"acapella", "acoustic", "demo", "ext", "extended", "inst", "instrumental", "live", "mix", "piano", "radio", "remix", "remixed", "ver", "version"},
		"Cyrillic": {"акустика", "версия", "инструментал", "радио"},
	}
	AltTrackExp *regexp.Regexp = regexp.MustCompile(
		strings.Join([]string{
			fmt.Sprintf(
				`(?i)\s+[-‐-―].*\PL(?:%[1]v)(?:\PL|$).*|\s+\p{Ps}(?:.*\PL)?(?:%[1]v)(?:\PL.*)?\p{Pe}`,
				strings.Join(slices.Concat(AltTrackTermGroups["Latin"], AltTrackTermGroups["Cyrillic"]), "|"),
			),
		}, "|"),
	)
	AlmostAltExp     *regexp.Regexp      = regexp.MustCompile(`\s+[-‐-―]\s+.*|\s*\p{Ps}.+\p{Pe}`)
	NotAltTermGroups map[string][]string = map[string][]string{
		"Latin": {"intro", "interlude"},
	}
	NotAltExp *regexp.Regexp = regexp.MustCompile(
		fmt.Sprintf(`\s+[-‐-―]\s+(?:%[1]v)$|\s*\p{Ps}(?:%[1]v)\p{Pe}$`, strings.Join(NotAltTermGroups["Latin"], "|")),
	)
)

type MGClient struct {
	MBClient   *mb2.Client
	MBLimitter *time.Ticker
}

func NewMGClient() (MGClient, func()) {
	client := MGClient{
		MBClient:   mb2.NewClient("musicgreed", "v0.1.0"),
		MBLimitter: time.NewTicker(time.Second),
	}
	return client, client.Stop
}

func (mgc MGClient) Stop() {
	mgc.MBLimitter.Stop()
}

func (mgc MGClient) MBTick() {
	<-mgc.MBLimitter.C
}

func ReleaseGroupsByArtist(client MGClient, artistID mb2.MBID, status string) ([]mb2.ReleaseGroup, error) {
	rgByMBID := make(map[mb2.MBID]mb2.ReleaseGroup)
	// Page through releases
	paginator := mb2.DefaultPaginator()
	rFilter := mb2.ReleaseFilter{ArtistMBID: artistID, Status: status, Includes: []string{"release-groups", "media", "recordings"}}
	for {
		client.MBTick()
		result, err := client.MBClient.BrowseReleases(rFilter, paginator)
		if err != nil || len(result.Releases) == 0 {
			if err != nil {
				fmt.Println(err)
			}
			break
		}
		for _, r := range result.Releases {
			rg, ok := rgByMBID[r.ReleaseGroup.ID]
			if ok {
				rg.Releases = append(rg.Releases, r)
			} else {
				rg = *r.ReleaseGroup
				rg.Releases = append(rg.Releases, r)
				rgByMBID[r.ReleaseGroup.ID] = rg
			}
		}
		paginator.Offset = paginator.Offset + len(result.Releases)
	}
	var groups []mb2.ReleaseGroup
	for _, v := range rgByMBID {
		groups = append(groups, v)
	}

	return groups, nil
}
