package cmd

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/frigorific44/musicgreed/concurrency"
	"github.com/frigorific44/musicgreed/musicinfo"
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

		discardTypes, _ := cmd.Flags().GetStringSlice("dsec")
		filtered := filteredReleases(groups, discardTypes)

		fmt.Println("Calculating set covers...")
		covers := setcovers(filtered)
		fmt.Println("---Set Covers---")
		for i, msc := range covers {
			contribution := contributions(msc)
			slices.SortFunc(contribution, func(a, b releaseContribution) int {
				return -1 * cmp.Compare(a.Contribution, b.Contribution)
			})
			fmt.Println("Set Cover", i)
			for rIndex, r := range msc {
				fmt.Printf("%v %v \n", contribution[rIndex].Contribution, r.Title)
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
	setcoverCmd.Flags().Bool("dalt", false, "discard")
}

func filteredReleases(groups []mb2.ReleaseGroup, dSecTypes []string) []mb2.Release {
	// Gather unique releases.
	var releases []mb2.Release
ReleaseGroupLoop:
	for _, rg := range groups {
		for _, dType := range dSecTypes {
			for _, sType := range rg.SecondaryTypes {
				if strings.EqualFold(dType, sType) {
					continue ReleaseGroupLoop
				}
			}
		}
		releases = append(releases, removeDuplicateReleases(rg.Releases)...)
	}
	return releases
}

func setcovers(releases []mb2.Release) [][]mb2.Release {
	trackMap := make(map[string][]int)
	for i, r := range releases {
		for _, t := range releaseTrackTitles(r) {
			tReleases, ok := trackMap[t]
			if ok {
				tReleases = append(tReleases, i)
				trackMap[t] = tReleases
			} else {
				trackMap[t] = []int{i}
			}
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

func removeDuplicateReleases(releases []mb2.Release) []mb2.Release {
	// Gather each release's track titles, sorted alphabetically.
	rTracks := make(map[int][]string)
	for i, r := range releases {
		rTracks[i] = releaseTrackTitles(r)
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

func contributions(setcover []mb2.Release) []releaseContribution {
	contributions := make([]releaseContribution, len(setcover))

	for i, release := range setcover {
		otherTracks := make(map[string]bool)
		for j, other := range setcover {
			if j == i {
				continue
			}
			tracks := releaseTrackTitles(other)
			for _, track := range tracks {
				otherTracks[track] = true
			}
		}
		tracks := releaseTrackTitles(release)
		var contribution int
		for _, track := range tracks {
			if _, ok := otherTracks[track]; !ok {
				contribution += 1
			}
		}
		contributions[i] = releaseContribution{Release: release, Contribution: contribution}
	}

	return contributions
}

func releaseTrackTitles(release mb2.Release) []string {
	var tracks []string
	for _, m := range release.Media {
		for _, t := range m.Tracks {
			tracks = append(tracks, t.Title)
		}
	}
	slices.Sort(tracks)
	return tracks
}
