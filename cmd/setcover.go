package cmd

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/frigorific44/musicgreed/beets"
	"github.com/frigorific44/musicgreed/concurrency"
	"github.com/frigorific44/musicgreed/musicinfo"
	"github.com/frigorific44/musicgreed/prompt"
	"github.com/spf13/cobra"
	mb2 "go.uploadedlobster.com/musicbrainzws2"
)

const (
	tableHeader string = "Contribution | Release(s)"
)

var (
	horizontal string = strings.Repeat("—", len(tableHeader))
)

// setcoverCmd represents the setcover command
func NewSetCoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   `setcover artist`,
		Short: "Compute the set cover for the complete song collection of an artist.",
		Long: "This command computes the minimal set of releases needed to contain every " +
			"unique track released by an artist. In addition, the unique contribution of " +
			"each release it output to assist a song collector's efforts. Available " +
			"flags may help to filter out music tracks that aren't of concern, depending " +
			"on desired thoroughness." +
			"\n\nTo discard live and remixed releases:" +
			"\n\n`musicgreed setcover --dsec=\"live,remix\" artist`" +
			"\n\nThe previous command can only discard whole releases tagged as " +
			"mentioned. To discard individual tracks that are parenthesized as an " +
			"alternate version:" +
			"\n\n`musicgreed setcover --dalt artist`" +
			"\n\nIf you maintain your library with the beets library manager, you can exclude " +
			"your collection from `setcover` with the remainder flag:" +
			"\n\n`musicgreed setcover -r artist`",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			scc := setCoverConfig{setCoverFlags: packageSetCoverFlags(cmd)}
			client, stop := musicinfo.NewMGClient()
			defer stop()

			mbid, idErr := artistMBID(client, args[0])
			if idErr != nil {
				fmt.Println("Artist ID could not be retrieved")
				return
			}
			scc.ArtistMBID = mbid

			fmt.Println("Retrieving music...")
			var status string
			if scc.Official {
				status = "official"
			}
			groups, err := musicinfo.ReleaseGroupsByArtist(client, scc.ArtistMBID, status)
			if err != nil {
				fmt.Println(err)
				return
			}

			// Pre-processing
			filtered := filterBySecondaryType(groups, scc)
			learnTracks(filtered, &scc)
			slog.Debug(
				"Set Cover Configuration",
				"Config", scc)
			// remove duplicates
			var releases []mb2.Release
			for _, rg := range filtered {
				releases = append(releases, uniqueReleases(rg.Releases, scc)...)
			}

			fmt.Println("Calculating set covers...")
			covers := setcovers(releases, scc)
			for i, msc := range covers {
				contribution := contributions(msc, scc)
				slices.SortFunc(contribution, func(a, b coverContribution) int {
					conComp := cmp.Compare(a.Contribution, b.Contribution)
					if conComp == 0 {
						return cmp.Compare(a.Title, b.Title)
					}
					return -1 * conComp
				})
				fmt.Print("\n> Set Cover ", i)
				fmt.Println(",", len(contribution), "releases")
				fmt.Println(horizontal)
				var titles []string
				fmt.Println(tableHeader)
				fmt.Println(horizontal)
				var currTitles []string
				for conI, c := range contribution {
					currTitles = append(currTitles, c.Title)
					titles = append(titles, c.Title)
					if conI+1 == len(contribution) || contribution[conI+1].Contribution != c.Contribution {
						fmt.Printf("%-14v %v\n", c.Contribution, strings.Join(currTitles, "; "))
						currTitles = nil
					}
				}
				fmt.Println("\nRelease Titles:")
				fmt.Println(strings.Join(titles, "; "))
				slog.Debug(
					fmt.Sprint("set cover result", i),
					"set cover", contribution)
			}
		},
	}

	cmd.Flags().StringSlice("dsec", []string{},
		"discard MusicBrainz secondary release group types (https://musicbrainz.org/doc/Release_Group/Type)",
	)
	cmd.Flags().Bool("dalt", false, "discard parenthesized alternate tracks (acoustic, remix, etc.)")
	cmd.Flags().Bool("official", false, "only official releases (https://musicbrainz.org/doc/Release#Status)")
	cmd.Flags().BoolP("remainder", "r", false, "requires a beets music library; calculates on the remainder after library tracks")

	return cmd
}

type setCoverFlags struct {
	DSec      []string
	DAlt      bool
	Official  bool
	Remainder bool
}

type setCoverConfig struct {
	setCoverFlags
	TitleSub    map[string]string
	TitleIgnore map[string]bool
	ArtistMBID  mb2.MBID
}

func packageSetCoverFlags(cmd *cobra.Command) setCoverFlags {
	dSec, _ := cmd.Flags().GetStringSlice("dsec")
	dAlt, _ := cmd.Flags().GetBool("dalt")
	official, _ := cmd.Flags().GetBool("official")
	remainder, _ := cmd.Flags().GetBool("remainder")
	return setCoverFlags{DSec: dSec, DAlt: dAlt, Official: official, Remainder: remainder}
}

func artistMBID(client musicinfo.MGClient, query string) (mb2.MBID, error) {
	if id := mb2.MBID(query); id.IsValid() {
		return id, nil
	} else {
		client.MBTick()
		res, err := client.MBClient.SearchArtists(mb2.SearchFilter{Query: query}, mb2.DefaultPaginator())
		if err != nil {
			return mb2.MBID(""), err
		} else {
			if len(res.Artists) > 0 {
				return res.Artists[0].ID, nil
			}
			return mb2.MBID(""), fmt.Errorf(`not a MBID and nothing returned from search`)
		}
	}
}

func filterBySecondaryType(groups []mb2.ReleaseGroup, scc setCoverConfig) []mb2.ReleaseGroup {
	var filtered []mb2.ReleaseGroup
ReleaseGroupLoop:
	for _, rg := range groups {
		for _, dType := range scc.DSec {
			for _, sType := range rg.SecondaryTypes {
				if strings.EqualFold(dType, sType) {
					continue ReleaseGroupLoop
				}
			}
		}
		filtered = append(filtered, rg)
	}
	return filtered
}

func setcovers(releases []mb2.Release, scc setCoverConfig) [][]mb2.Release {
	trackMap := make(map[string][]int)
	for i, r := range releases {
		for _, t := range releaseTrackTitles(r, scc) {
			trackMap[t] = append(trackMap[t], i)
		}
	}

	combinations := minimalCombinations(trackMap)

	var covers [][]mb2.Release
	for _, p := range combinations {
		var sc []mb2.Release
		for _, i := range p {
			sc = append(sc, releases[i])
		}
		covers = append(covers, sc)
	}
	return covers
}

func uniqueReleases(releases []mb2.Release, scc setCoverConfig) []mb2.Release {
	// Gather each release's track titles, sorted alphabetically.
	rTracks := make(map[int][]string)
	for i, r := range releases {
		rTracks[i] = releaseTrackTitles(r, scc)
	}
	var groups [][]mb2.Release
	// Each loop, form a group of releases with identical track titles.
	for len(rTracks) > 0 {
		ref := -1
		var refVal []string
		var group []mb2.Release
		for k, v := range rTracks {
			if ref < 0 {
				ref = k
				refVal = v
			} else {
				if len(refVal) != len(v) {
					continue
				}
				isEqual := true
				for i, t := range refVal {
					if t != v[i] {
						isEqual = false
						break
					}
				}
				if isEqual {
					group = append(group, releases[k])
					delete(rTracks, k)
				}
			}
		}
		group = append(group, releases[ref])
		delete(rTracks, ref)
		groups = append(groups, group)
	}
	var toReturn []mb2.Release
	// Select one release to reperesent each group.
	// TODO: Way to set release region preference, or to somehow collate titles the releases may be known under
	for _, g := range groups {
		toReturn = append(toReturn, g[0])
	}
	return toReturn
}

func minimalCombinations(trackMap map[string][]int) [][]int {
	var combinations [][]int
	var wg sync.WaitGroup
	var minima concurrency.RWInt
	minima.Set(int(^uint(0) >> 1))
	comboChan := make(chan []int)

	wg.Add(1)
	go minCombosRecursive(trackMap, []int{}, &minima, &wg, comboChan)
	go func() {
		wg.Wait()
		close(comboChan)
	}()
	for c := range comboChan {
		if len(c) > minima.Get() {
			continue
		}
		if len(c) < minima.Get() {
			combinations = nil
			minima.Set(len(c))
		}
		combinations = append(combinations, c)
	}

	return combinations
}

func minCombosRecursive(
	trackMap map[string][]int,
	curr []int,
	minima *concurrency.RWInt,
	wg *sync.WaitGroup,
	results chan<- []int,
) {
	defer wg.Done()
	if len(trackMap) > 0 {
		if len(curr) == minima.Get() {
			return
		}

		// Retrieve the rarest entry to permute on.
		keys := make([]string, len(trackMap))
		i := 0
		for k := range trackMap {
			keys[i] = k
			i++
		}
		slices.SortFunc(keys, func(a, b string) int {
			return cmp.Compare(len(trackMap[a]), len(trackMap[b]))
		})
		value := trackMap[keys[0]]

		// Combine on the releases for selected entry
		for _, r := range value {
			newCurr := make([]int, len(curr))
			copy(newCurr, curr)
			newCurr = append(newCurr, r)
			// Copy map
			newMap := make(map[string][]int)
			for k, v := range trackMap {
				newMap[k] = v
			}
			// Remove pairs which include release in their value
			for k, v := range newMap {
				for _, i := range v {
					if r == i {
						delete(newMap, k)
						break
					}
				}
			}
			wg.Add(1)
			go minCombosRecursive(newMap, newCurr, minima, wg, results)
		}
	} else {
		results <- curr
	}
}

type coverContribution struct {
	Title        string
	ID           mb2.MBID
	Tracks       []string
	Contribution int
}

func contributions(setcover []mb2.Release, scc setCoverConfig) []coverContribution {
	contributions := make([]coverContribution, len(setcover))

	for i, release := range setcover {
		otherTracks := make(map[string]bool)
		for j, other := range setcover {
			if j == i {
				continue
			}
			tracks := releaseTrackTitles(other, scc)
			for _, track := range tracks {
				otherTracks[track] = true
			}
		}
		tracks := releaseTrackTitles(release, scc)
		var contribution int
		for _, track := range tracks {
			if !otherTracks[track] {
				contribution += 1
			}
		}
		contributions[i] = coverContribution{
			Title:        release.Title,
			ID:           release.ID,
			Tracks:       tracks,
			Contribution: contribution}
	}

	return contributions
}

func releaseTrackTitles(release mb2.Release, scc setCoverConfig) []string {
	var tracks []string
	for _, m := range release.Media {
		for _, t := range m.Tracks {
			if scc.TitleIgnore[t.Title] || t.Recording.IsVideo {
				continue
			}
			if sub, ok := scc.TitleSub[t.Title]; ok {
				tracks = append(tracks, sub)
			} else {
				tracks = append(tracks, t.Title)
			}
		}
	}
	slices.Sort(tracks)
	return tracks
}

// Cleans the title for comparability without changing the meaning
func CleanTitle(title string) string {
	title = strings.ToLower(title)
	title = strings.ReplaceAll(title, "&", "and")
	return strings.Map(func(r rune) rune {
		switch r {
		case '’':
			return '\''
		case ',':
			return -1
		}
		return r
	}, title)
}

// Embeds title substitutions (whens tracks are the same but titled differently),
// as well as tracks to ignore into the configuration.
func learnTracks(groups []mb2.ReleaseGroup, scc *setCoverConfig) {
	subSets := make(map[string]map[string]bool)
	ignore := make(map[string]bool)

	// libraryIDs := make(map[mb2.MBID]bool)
	if scc.Remainder {
		beetsTracks, beetErr := beets.ArtistTrackTitles(scc.ArtistMBID)
		slog.Debug(
			"current beets library",
			"ArtistID", scc.ArtistMBID,
			"Error", beetErr,
			"Size", len(beetsTracks))
		for _, t := range beetsTracks {
			// libraryIDs[t.ID] = true
			ignore[t.Title] = true
		}
	}

	titleSet := make(map[string]bool)
	for _, rg := range groups {
		for _, r := range rg.Releases {
			for _, m := range r.Media {
				for _, t := range m.Tracks {
					titleSet[t.Title] = true
				}
			}
		}
	}

	metric := metrics.NewLevenshtein()
	metric.CaseSensitive = false
	altTracks := make(map[string]bool)

	// Process alternate tracks.
	// Instead, check for existence of root in title set
	for t := range titleSet {
		var manual bool
		if musicinfo.AlmostAltExp.MatchString(t) {
			root := musicinfo.AlmostAltExp.ReplaceAllLiteralString(t, "")
			if musicinfo.AltTrackExp.MatchString(t) && titleSet[root] {
				altTracks[t] = true
				manual = false
			} else if prompt.BoolPrompt(fmt.Sprint("Is this an alternate track: ", t), true) {
				altTracks[t] = true
				manual = true
			}
		}
		if altTracks[t] && scc.DAlt {
			slog.Debug(
				"track marked as an alternate",
				"title", t,
				"manual", manual)
			ignore[t] = true
			delete(titleSet, t)
		}
	}

	for t := range titleSet {
		for other := range titleSet {
			if t == other || (altTracks[t] != altTracks[other]) {
				continue
			}
			if strutil.Similarity(t, other, metric) > 0.6 {
				if altTracks[t] && altTracks[other] {
					rootA := musicinfo.AltTrackExp.ReplaceAllLiteralString(t, "")
					rootB := musicinfo.AltTrackExp.ReplaceAllLiteralString(other, "")
					if strutil.Similarity(rootA, rootB, metric) <= 0.5 {
						continue
					}
				}
				if CleanTitle(t) != CleanTitle(other) && !prompt.BoolPrompt(fmt.Sprintf(`Are tracks "%v" and "%v" equal?`, t, other), true) {
					continue
				}
				m1, ok1 := subSets[t]
				m2, ok2 := subSets[other]
				if ok1 && ok2 {
					// Merge sets
					for k := range m2 {
						m1[k] = true
					}
					subSets[other] = m1
				} else if ok1 {
					m1[other] = true
					subSets[other] = m1
				} else if ok2 {
					m2[t] = true
					subSets[t] = m2
				} else {
					m := map[string]bool{t: true, other: true}
					subSets[t] = m
					subSets[other] = m
				}
			}
		}
		delete(titleSet, t)
	}

	sub := make(map[string]string)
	for _, m := range subSets {
		slog.Debug(
			"titles determined to be equivalent",
			"set", m)
		set := make([]string, 0, len(m))
		var ignored bool
		for el := range m {
			set = append(set, el)
			if ignore[el] {
				ignored = true
			}
		}
		slices.SortFunc(set, func(a, b string) int {
			return -1 * cmp.Compare(len(a), len(b))
		})
		if ignored {
			ignore[set[0]] = true
		}
		for _, el := range set {
			sub[el] = set[0]
		}
	}

	scc.TitleIgnore = ignore
	scc.TitleSub = sub
}
