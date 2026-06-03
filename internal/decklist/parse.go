// Package decklist parses the JTLA jumpstart decklist format and groups
// cards by theme for label rendering.
package decklist

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Card represents a single card entry from the decklist.
type Card struct {
	Qty    int
	Name   string
	Set    string
	Number int
	Themes []string
}

// lineRe matches: <qty> <name> (<SET>) <number><rest>
var lineRe = regexp.MustCompile(`^(\d+) (.+?) \(([A-Z]+)\) (\d+)(.*)$`)

// basicLands is the set of MTG basic land names.
var basicLands = map[string]bool{
	"Forest": true, "Island": true, "Mountain": true,
	"Plains": true, "Swamp": true,
}

// IsBasic reports whether the card is a basic land.
func IsBasic(c Card) bool {
	return basicLands[c.Name]
}

// ParseReader reads all card lines from r and returns the parsed cards.
// Lines that don't match the expected format are silently skipped.
func ParseReader(r io.Reader) ([]Card, error) {
	var cards []Card
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		card, err := parseLine(line)
		if err != nil {
			continue
		}
		cards = append(cards, *card)
	}
	return cards, scanner.Err()
}

func parseLine(line string) (*Card, error) {
	m := lineRe.FindStringSubmatch(line)
	if m == nil {
		return nil, fmt.Errorf("no match: %q", line)
	}
	qty, _ := strconv.Atoi(m[1])
	name := strings.TrimSpace(m[2])
	set := m[3]
	num, _ := strconv.Atoi(m[4])

	var themes []string
	rest := strings.TrimSpace(m[5])
	if strings.HasPrefix(rest, "#") {
		rest = rest[1:] // strip leading "#"
		for _, t := range strings.Split(rest, " #") {
			if t = strings.TrimSpace(t); t != "" {
				themes = append(themes, t)
			}
		}
	}

	return &Card{Qty: qty, Name: name, Set: set, Number: num, Themes: themes}, nil
}

// AllThemes returns every unique theme name found in cards, sorted.
func AllThemes(cards []Card) []string {
	seen := make(map[string]struct{})
	for _, c := range cards {
		for _, t := range c.Themes {
			seen[t] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// ThemeCards returns all cards that belong to theme: non-basics sorted
// alphabetically by name first, then basics sorted by qty desc then name.
func ThemeCards(cards []Card, theme string) []Card {
	var nonbasics, basics []Card
	for _, c := range cards {
		for _, t := range c.Themes {
			if t == theme {
				if IsBasic(c) {
					basics = append(basics, c)
				} else {
					nonbasics = append(nonbasics, c)
				}
				break
			}
		}
	}
	sort.SliceStable(nonbasics, func(i, j int) bool {
		return nonbasics[i].Name < nonbasics[j].Name
	})
	sort.SliceStable(basics, func(i, j int) bool {
		if basics[i].Qty != basics[j].Qty {
			return basics[i].Qty > basics[j].Qty
		}
		return basics[i].Name < basics[j].Name
	})
	return append(nonbasics, basics...)
}
