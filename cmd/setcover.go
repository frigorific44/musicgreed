package cmd

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/frigorific44/musicgreed/concurrency"
	"github.com/frigorific44/musicgreed/musicinfo"
	"github.com/frigorific44/musicgreed/prompt"
	"github.com/spf13/cobra"
	mb2 "go.uploadedlobster.com/musicbrainzws2"
)

// setcoverCmd represents the setcover command
var setcoverCmd = &cobra.Command{
	Use:   `setcover "MBID"`,
	Short: "Compute the set cover for the complete song collection of an artist.",
	Long: `This command compute the minimal set of releases needed in a collection
to contain every unique track released by an artist. In addition, the unique 
contribution of each release it output to to assist a song collector's efforts.
Available flags may help to filter out music tracks that aren't of concern, 
depending on desired thoroughness:

To discard live and remixed releases:
musicgreed setcover "MBID" --dsec="live,remix"

The previous command can only discard whole releases tagged as mentioned. To
discard individual tracks that are parenthesized as an alternate version:
musicgreed setcover "MBID" --dalt`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Ensure any positional arguments correspond to a MBID
		re := regexp.MustCompile(`^[A-Fa-f0-9]{8}(-[A-Fa-f0-9]{4}){3}-[A-Fa-f0-9]{12}$`)
		for _, a := range args {
			if !re.MatchString(a) {
				return fmt.Errorf(`value does not resemble MBID: %v`, a)
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		mbid := mb2.MBID(args[0])
		client, stop := musicinfo.NewMGClient()
		defer stop()
		fmt.Println("Retrieving music...")
		groups, err := musicinfo.ReleaseGroupsByArtist(client, mbid)
		if err != nil {
			fmt.Println(err)
			return
		}

		scc := setCoverConfig{setCoverFlags: packageSetCoverFlags(cmd)}
		learnTracks(groups, &scc)
		filtered := filteredReleases(groups, scc)

		fmt.Println("Calculating set covers...")
		covers := setcovers(filtered, scc)
		fmt.Println("---Set Covers---")
		for i, msc := range covers {
			contribution := contributions(msc, scc)
			slices.SortFunc(contribution, func(a, b releaseContribution) int {
				return -1 * cmp.Compare(a.Contribution, b.Contribution)
			})
			fmt.Println("Set Cover", i)
			for _, c := range contribution {
				fmt.Printf("%v %v \n", c.Contribution, c.Title)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(setcoverCmd)

	// Local flags.
	setcoverCmd.Flags().StringSlice("dsec", []string{},
		"discard MusicBrainz secondary release group types (https://musicbrainz.org/doc/Release_Group/Type)",
	)
	setcoverCmd.Flags().Bool("dalt", false, "discard parenthesized alternate tracks (acoustic, remix, etc.)")
}

type setCoverFlags struct {
	DSec []string
	DAlt bool
}

type setCoverConfig struct {
	setCoverFlags
	TitleSub    map[string]string
	TitleIgnore map[string]bool
}

func packageSetCoverFlags(cmd *cobra.Command) setCoverFlags {
	dSec, _ := cmd.Flags().GetStringSlice("dsec")
	dAlt, _ := cmd.Flags().GetBool("dalt")
	return setCoverFlags{DSec: dSec, DAlt: dAlt}
}

func filteredReleases(groups []mb2.ReleaseGroup, scc setCoverConfig) []mb2.Release {
	// Gather unique releases.
	var releases []mb2.Release
ReleaseGroupLoop:
	for _, rg := range groups {
		for _, dType := range scc.DSec {
			for _, sType := range rg.SecondaryTypes {
				if strings.EqualFold(dType, sType) {
					continue ReleaseGroupLoop
				}
			}
		}
		releases = append(releases, removeDuplicateReleases(rg.Releases, scc)...)
	}
	return releases
}

func setcovers(releases []mb2.Release, scc setCoverConfig) [][]mb2.Release {
	trackMap := make(map[string][]int)
	for i, r := range releases {
		for _, t := range releaseTrackTitles(r, scc) {
			trackMap[t] = append(trackMap[t], i)
		}
	}

	combinations := coverCombinations(trackMap)
	minima := int(^uint(0) >> 1)
	for _, p := range combinations {
		if len(p) < minima {
			minima = len(p)
		}
	}

	var minsetcovers [][]mb2.Release
	for _, p := range combinations {
		if len(p) == minima {
			var sc []mb2.Release
			for _, i := range p {
				sc = append(sc, releases[i])
			}
			minsetcovers = append(minsetcovers, sc)
		}
	}
	return minsetcovers
}

func removeDuplicateReleases(releases []mb2.Release, scc setCoverConfig) []mb2.Release {
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
	for _, g := range groups {
		toReturn = append(toReturn, g[0])
	}
	return toReturn
}

func coverCombinations(trackMap map[string][]int) [][]int {
	var combinations [][]int
	var wg sync.WaitGroup
	var minima concurrency.RWInt
	minima.Set(int(^uint(0) >> 1))
	permChan := make(chan []int)
	wg.Add(1)
	go covPerRecursive(trackMap, []int{}, &minima, &wg, permChan)
	go func() {
		wg.Wait()
		close(permChan)
	}()
	for p := range permChan {
		combinations = append(combinations, p)
	}
	return combinations
}

func covPerRecursive(trackMap map[string][]int, curr []int, minima *concurrency.RWInt, wg *sync.WaitGroup, res chan<- []int) {
	defer wg.Done()
	if len(trackMap) > 0 {
		if len(curr) == minima.Get() {
			return
		}
		var value []int
		// Retrieve a pair to permute on.
		for _, value = range trackMap {
			break
		}
		for _, r := range value {
			newCurr := curr
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
			go covPerRecursive(newMap, newCurr, minima, wg, res)
		}
	} else {
		if len(curr) < minima.Get() {
			minima.Set(len(curr))
		}
		res <- curr
	}
}

type releaseContribution struct {
	mb2.Release
	Contribution int
}

func contributions(setcover []mb2.Release, scc setCoverConfig) []releaseContribution {
	contributions := make([]releaseContribution, len(setcover))

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
		contributions[i] = releaseContribution{Release: release, Contribution: contribution}
	}

	return contributions
}

func releaseTrackTitles(release mb2.Release, scc setCoverConfig) []string {
	var tracks []string
	for _, m := range release.Media {
		for _, t := range m.Tracks {
			if scc.TitleIgnore[t.Title] {
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

// Embeds title substitutions (whens tracks are the same but titled differently),
// as well as tracks to ignore into the configuration.
func learnTracks(groups []mb2.ReleaseGroup, scc *setCoverConfig) {
	subSets := make(map[string]map[string]bool)
	ignore := make(map[string]bool)

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
	altExp := regexp.MustCompile(`(?i).*\(.*\b(?:live|mix|version|remix|extended|ver\.|ext\.|acoustic|piano)\b.*\).*`)
	almostAltExp := regexp.MustCompile(`.*\(.+\).*`)
	altTracks := make(map[string]bool)

	for t := range titleSet {
		if altExp.MatchString(t) || (almostAltExp.MatchString(t) && prompt.BoolPrompt(fmt.Sprint("Is this an alternate track: ", t), true)) {
			altTracks[t] = true
			if scc.DAlt {
				ignore[t] = true
				delete(titleSet, t)
			}
		}
	}

	for t := range titleSet {
		for other := range titleSet {
			if t == other || (altTracks[t] != altTracks[other]) {
				continue
			}
			if strutil.Similarity(t, other, metric) > 0.5 {
				// If only one title contains alt keywords
				//// If DAlt, discard alt (with confirmation?)
				//// Else, keep both
				// Else, ask if the same.
				if prompt.BoolPrompt(fmt.Sprintf(`Are tracks "%v" and "%v" equal?`, t, other), true) {
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
		}
		delete(titleSet, t)
	}

	sub := make(map[string]string)
	for _, m := range subSets {
		set := make([]string, 0, len(m))
		for el := range m {
			set = append(set, el)
		}
		slices.SortFunc(set, func(a, b string) int {
			return -1 * cmp.Compare(len(a), len(b))
		})
		for _, el := range set {
			sub[el] = set[0]
		}
	}

	scc.TitleIgnore = ignore
	scc.TitleSub = sub
}
