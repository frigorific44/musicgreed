package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func StringPrompt(label string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, label+" ")
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	return strings.TrimSpace(s)
}

func IntPrompt(label string) int {
	for {
		s := StringPrompt(label)
		i, err := strconv.Atoi(s)
		if err == nil {
			return i
		}
	}
}
