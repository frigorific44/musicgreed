package cmd

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/frigorific44/musicgreed/musicinfo"
	"github.com/spf13/cobra"
	mb2 "go.uploadedlobster.com/musicbrainzws2"
)

// setcoverCmd represents the setcover command
var setcoverCmd = &cobra.Command{
	Use:   `setcover "MBID"`,
	Short: "Compute the set cover for the complete song collection of an artist.",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
		groups, err := musicinfo.ReleaseGroupsByArtist(client, mbid)
		if err != nil {
			fmt.Println(err)
			return
		}
		covers := setcovers(groups)
		for i, msc := range covers {
			contribution := calculateContributions(msc)
			fmt.Println("---Set Covers---")
			fmt.Println(i)
			for rIndex, r := range msc {
				fmt.Printf("%v %v \n", contribution[rIndex], r.Title)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(setcoverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// setcoverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// setcoverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func setcovers(groups []mb2.ReleaseGroup) [][]mb2.Release {
	var releases []mb2.Release

	for _, rg := range groups {
		releases = append(releases, removeDuplicateReleases(rg.Releases)...)
	}

	trackMap := make(map[string][]int)
	for i, r := range releases {
		for _, m := range r.Media {
			for _, t := range m.Tracks {
				tReleases, ok := trackMap[t.Title]
				if ok {
					tReleases = append(tReleases, i)
					trackMap[t.Title] = tReleases
				} else {
					trackMap[t.Title] = []int{i}
				}
			}
		}
	}

	permutations := coverPermutations(trackMap, []int{})
	minima := int(^uint(0) >> 1)
	for _, p := range permutations {
		if len(p) < minima {
			minima = len(p)
		}
	}

	var minsetcovers [][]mb2.Release
	for _, p := range permutations {
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
		var tracks []string
		for _, m := range r.Media {
			for _, t := range m.Tracks {
				tracks = append(tracks, t.Title)
			}
		}
		slices.Sort(tracks)
		rTracks[i] = tracks
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

func coverPermutations(trackMap map[string][]int, curr []int) [][]int {
	var permutations [][]int

	if len(trackMap) > 0 {
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
			newPermutations := coverPermutations(newMap, newCurr)
			permutations = append(permutations, newPermutations...)
		}
	} else {
		permutations = append(permutations, curr)
	}

	return permutations
}

func calculateContributions(setcover []mb2.Release) []int {
	contributions := make([]int, len(setcover))

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
		contributions[i] = contribution
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
