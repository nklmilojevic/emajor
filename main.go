package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed data/emojis.json
var emojiDataJSON []byte

type Emoji struct {
	Char     string   `json:"c"`
	Name     string   `json:"n"`
	Alias    string   `json:"a"`
	Keywords []string `json:"k"`
	Group    string   `json:"g"`
}

var allEmojis []Emoji

func init() {
	if err := json.Unmarshal(emojiDataJSON, &allEmojis); err != nil {
		panic(err)
	}
}

// ── Alfred types ─────────────────────────────────────────────────────────────

type alfredMod struct {
	Arg      string `json:"arg"`
	Subtitle string `json:"subtitle"`
	Valid    bool   `json:"valid"`
}

type alfredText struct {
	Copy      string `json:"copy"`
	LargeType string `json:"largetype"`
}

type alfredIcon struct {
	Path string `json:"path"`
}

type alfredItem struct {
	UID          string               `json:"uid"`
	Title        string               `json:"title"`
	Subtitle     string               `json:"subtitle,omitempty"`
	Arg          string               `json:"arg"`
	Autocomplete string               `json:"autocomplete,omitempty"`
	Valid        *bool                `json:"valid,omitempty"`
	Text         alfredText           `json:"text"`
	Icon         *alfredIcon          `json:"icon,omitempty"`
	Mods         map[string]alfredMod `json:"mods,omitempty"`
}

type alfredOutput struct {
	Items []alfredItem `json:"items"`
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: emajor <search|use> [args]")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "search":
		query := ""
		if len(os.Args) > 2 {
			query = strings.Join(os.Args[2:], " ")
		}
		cmdSearch(query)
	case "use":
		if len(os.Args) < 3 {
			os.Exit(1)
		}
		cmdUse(os.Args[2])
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(1)
	}
}

// ── Search ────────────────────────────────────────────────────────────────────

// sharedPrefixLen returns the number of equal leading bytes in a and b.
func sharedPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// stemMatch returns true when tok and kw share the same word stem even if
// neither is a prefix of the other (e.g. "celebrate" ↔ "celebration").
// Requires at least 4 shared chars and ≥80% overlap of the shorter string.
func stemMatch(tok, kw string) bool {
	if len(tok) < 4 || len(kw) < 4 {
		return false
	}
	shared := sharedPrefixLen(tok, kw)
	minLen := min(len(tok), len(kw))
	return shared >= 4 && shared*10 >= minLen*8
}

// prefixMatchOK returns true when tok is long enough to safely use as a
// keyword prefix (avoids "dog" matching "dogeza" style false positives).
func prefixMatchOK(tok string) bool { return len(tok) >= 4 }

// tokenScore returns how well a single token matches an emoji (0 = no match).
func tokenScore(e *Emoji, tok string) int {
	best := 0
	if e.Name == tok {
		best = max(best, 100)
	}
	for _, word := range strings.Fields(e.Name) {
		switch {
		case word == tok:
			best = max(best, 80)
		case prefixMatchOK(tok) && strings.HasPrefix(word, tok):
			best = max(best, 50)
		case stemMatch(tok, word):
			best = max(best, 40)
		case prefixMatchOK(tok) && strings.Contains(word, tok):
			best = max(best, 25)
		}
	}
	switch {
	case e.Alias == tok:
		best = max(best, 70)
	case prefixMatchOK(tok) && strings.HasPrefix(e.Alias, tok):
		best = max(best, 40)
	case stemMatch(tok, e.Alias):
		best = max(best, 30)
	case prefixMatchOK(tok) && strings.Contains(e.Alias, tok):
		best = max(best, 18)
	}
	for _, kw := range e.Keywords {
		switch {
		case kw == tok:
			best = max(best, 70)
		case prefixMatchOK(tok) && strings.HasPrefix(kw, tok):
			best = max(best, 45)
		case stemMatch(tok, kw):
			best = max(best, 35)
		case prefixMatchOK(tok) && strings.Contains(kw, tok):
			best = max(best, 15)
		}
	}
	return best
}

type searchResult struct {
	e     *Emoji
	score float64
}

// searchEmojis returns ranked emoji results for the given query.
func searchEmojis(query string) []*Emoji {
	query = strings.ToLower(strings.TrimSpace(query))
	tokens := strings.Fields(query)

	recents := loadRecent()
	recentRank := make(map[string]int, len(recents))
	for i, r := range recents {
		recentRank[r] = len(recents) - i
	}

	if len(tokens) == 0 {
		return emptyQueryEmojis(recents)
	}

	type rawRow struct {
		e      *Emoji
		scores []int
	}
	rows := make([]rawRow, len(allEmojis))
	tokenFreq := make([]int, len(tokens))

	for i := range allEmojis {
		rows[i].e = &allEmojis[i]
		rows[i].scores = make([]int, len(tokens))
		for t, tok := range tokens {
			s := tokenScore(&allEmojis[i], tok)
			rows[i].scores[t] = s
			if s > 0 {
				tokenFreq[t]++
			}
		}
	}

	N := float64(len(allEmojis))
	idf := make([]float64, len(tokens))
	for t, freq := range tokenFreq {
		if freq > 0 {
			idf[t] = math.Log(N/float64(freq)) + 1
		}
	}

	type scored struct {
		e     *Emoji
		score float64
	}
	results := make([]scored, 0, 200)
	for _, row := range rows {
		var rawScore float64
		matched := 0
		for t := range tokens {
			if row.scores[t] > 0 {
				rawScore += float64(row.scores[t]) * idf[t]
				matched++
			}
		}
		if rawScore == 0 {
			continue
		}
		completeness := float64(matched) / float64(len(tokens))
		score := rawScore*completeness*completeness + float64(recentRank[row.e.Char])*20
		results = append(results, scored{row.e, score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	if len(results) > 40 {
		results = results[:40]
	}

	out := make([]*Emoji, len(results))
	for i, r := range results {
		out[i] = r.e
	}
	return out
}

// emptyQueryEmojis returns only recently used emojis (most recent first).
func emptyQueryEmojis(recents []string) []*Emoji {
	out := make([]*Emoji, 0, len(recents))
	for _, r := range recents {
		for i := range allEmojis {
			if allEmojis[i].Char == r {
				out = append(out, &allEmojis[i])
				break
			}
		}
	}
	return out
}

func makeItem(e *Emoji) alfredItem {
	shortcode := ":" + e.Alias + ":"
	name := strings.ToUpper(e.Name[:1]) + e.Name[1:]
	return alfredItem{
		UID:          e.Alias,
		Title:        name,
		Arg:          e.Char,
		Autocomplete: e.Alias,
		Text:         alfredText{Copy: e.Char, LargeType: e.Char},
		Icon:         &alfredIcon{Path: iconCachePath(e.Alias)},
		Mods: map[string]alfredMod{
			"cmd": {Arg: shortcode, Subtitle: "Copy " + shortcode, Valid: true},
		},
	}
}

func cmdSearch(query string) {
	emojis := searchEmojis(query)

	var items []alfredItem
	if len(emojis) == 0 {
		f := false
		msg := "No emoji found for \"" + query + "\""
		if query == "" {
			msg = "Start typing to search for emojis"
		}
		items = []alfredItem{{
			Title: msg,
			Valid: &f,
		}}
	} else {
		ensureIcons(emojis)
		items = make([]alfredItem, len(emojis))
		for i, e := range emojis {
			items[i] = makeItem(e)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.Encode(alfredOutput{Items: items})
}

// ── Recent tracking ───────────────────────────────────────────────────────────

const maxRecent = 20

func recentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "emajor", "recent.json")
}

func loadRecent() []string {
	data, err := os.ReadFile(recentPath())
	if err != nil {
		return nil
	}
	var recents []string
	json.Unmarshal(data, &recents) //nolint:errcheck
	return recents
}

// cmdUse records the emoji as recently used and prints arg for Alfred piping.
// arg may be an emoji char ("😀") or a shortcode (":grinning:") — in both
// cases we resolve to the char for the recents list, then print arg as-is so
// Alfred pipes it unchanged to Copy to Clipboard.
func cmdUse(arg string) {
	char := arg
	if strings.HasPrefix(arg, ":") && strings.HasSuffix(arg, ":") && len(arg) > 2 {
		alias := arg[1 : len(arg)-1]
		for i := range allEmojis {
			if allEmojis[i].Alias == alias {
				char = allEmojis[i].Char
				break
			}
		}
	}

	recents := loadRecent()
	updated := make([]string, 0, len(recents)+1)
	updated = append(updated, char)
	for _, r := range recents {
		if r != char {
			updated = append(updated, r)
		}
	}
	if len(updated) > maxRecent {
		updated = updated[:maxRecent]
	}
	path := recentPath()
	os.MkdirAll(filepath.Dir(path), 0o755) //nolint:errcheck
	data, _ := json.Marshal(updated)
	os.WriteFile(path, data, 0o644) //nolint:errcheck
	fmt.Print(arg) // print original arg (emoji or shortcode) for clipboard
}
