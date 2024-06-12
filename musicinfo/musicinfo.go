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

func BuildArtist(client MGClient, mbid mb2.MBID) (mb2.Artist, error) {
	client.MBTick()
	artist, aErr := client.MBClient.LookupArtist(mbid, mb2.IncludesFilter{})
	if aErr != nil {
		return artist, aErr
	}

	// Page through release groups
	paginator := mb2.DefaultPaginator()
	rgFilter := mb2.ReleaseGroupFilter{ArtistMBID: mbid}
	for {
		client.MBTick()
		result, err := client.MBClient.BrowseReleaseGroups(rgFilter, paginator)
		if err != nil || len(result.ReleaseGroups) == 0 {
			if err != nil {
				fmt.Println(err)
			}
			break
		}
		artist.ReleaseGroups = append(artist.ReleaseGroups, result.ReleaseGroups...)
		paginator = mb2.Paginator{Limit: paginator.Limit, Offset: paginator.Offset + result.Count}
	}
	// Index release groups for adding releases
	rgByMBID := make(map[mb2.MBID]int)
	for i, rg := range artist.ReleaseGroups {
		rgByMBID[rg.ID] = i
	}
	// Page through releases
	paginator = mb2.DefaultPaginator()
	rFilter := mb2.ReleaseFilter{ArtistMBID: mbid, Includes: []string{"release-groups", "media", "recordings"}}
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
			rgIndex, ok := rgByMBID[r.ReleaseGroup.ID]
			if ok {
				artist.ReleaseGroups[rgIndex].Releases = append(artist.ReleaseGroups[rgIndex].Releases, r)
			}
		}
		paginator = mb2.Paginator{Limit: paginator.Limit, Offset: paginator.Offset + result.Count}
	}

	return artist, nil
}
