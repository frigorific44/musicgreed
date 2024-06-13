package musicinfo

import (
	"fmt"
	"time"

	mb2 "go.uploadedlobster.com/musicbrainzws2"
)

type MGClient struct {
	MBClient   *mb2.Client
	MBLimitter *time.Ticker
}

func NewMGClient() (MGClient, func()) {
	client := MGClient{
		MBClient:   mb2.NewClient("musicgreed", "0.1"),
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

func ReleaseGroupsByArtist(client MGClient, artistID mb2.MBID) ([]mb2.ReleaseGroup, error) {
	rgByMBID := make(map[mb2.MBID]mb2.ReleaseGroup)
	// Page through releases
	paginator := mb2.DefaultPaginator()
	rFilter := mb2.ReleaseFilter{ArtistMBID: artistID, Includes: []string{"release-groups", "media", "recordings"}}
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
