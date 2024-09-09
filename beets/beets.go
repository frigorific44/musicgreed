package beets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mb2 "go.uploadedlobster.com/musicbrainzws2"
)

const (
	lengthLayout    string = "4:05"
	trackFormatBase string = `{"id":%q,"title":%q,"length_str":%q,"position_str":%q}`
)

var (
	clockZero, _        = time.Parse(lengthLayout, "0:00")
	trackFormat  string = fmt.Sprintf(trackFormatBase, "$mb_releasetrackid", "$title", "$length", "$track")
)

func ArtistTrackTitles(id mb2.MBID) ([]mb2.Track, error) {
	var tracks []mb2.Track
	if _, err := exec.LookPath("beet"); err != nil {
		return tracks, fmt.Errorf(`beet executable not found: %w`, err)
	}
	cmdStr := fmt.Sprintf(`ls -f '%s' mb_artistids:%s`, trackFormat, id)
	cmd := exec.Command("beet", strings.Split(cmdStr, " ")...)
	out, err := cmd.Output()
	if err != nil {
		return tracks, fmt.Errorf(`command "%v" did not output cleanly: %w`, strings.Join(cmd.Args, " "), err)
	}
	tracks, err = unmarshalBeetsTracks(string(out))
	if err != nil {
		err = fmt.Errorf(`command "%v" did not unmarshal cleanly, %w`, strings.Join(cmd.Args, " "), err)
	}
	return tracks, err
}

func unmarshalBeetsTracks(beetStr string) ([]mb2.Track, error) {
	var tracks []mb2.Track
	var combined error
	for _, line := range strings.Split(string(beetStr), "\n") {
		if line == "" {
			continue
		}
		line = strings.Trim(line, "'")
		var inter intermediateTrack
		if err := json.Unmarshal([]byte(line), &inter); err != nil {
			combined = errors.Join(combined, fmt.Errorf(`problem unmarshaling line "%v": %w`, line, err))
		} else {
			inter.Length = parseLength(inter.LengthStr)
			inter.Position = parsePosition(inter.PositionStr)
			tracks = append(tracks, inter.Track)
		}
	}
	return tracks, combined
}

type intermediateTrack struct {
	mb2.Track
	LengthStr   string `json:"length_str"`
	PositionStr string `json:"position_str"`
}

func parseLength(dur string) mb2.Duration {
	t, _ := time.Parse("4:05", dur)
	return mb2.Duration{Duration: t.Sub(clockZero)}
}

func parsePosition(pos string) int {
	i, _ := strconv.Atoi(pos)
	return i
}
