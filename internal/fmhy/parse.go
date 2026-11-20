// Package fmhy reads the FreeMediaHeckYeah wiki and turns it into a list of
// sources kamui can search.
//
// FMHY is a plain-markdown wiki on GitHub. Each page is a list of sections,
// and each section is a list of bullet points holding links to websites.
// This package is only concerned with reading that text. Deciding which
// sources to actually use is the caller's job.
package fmhy

import (
	"net/url"
	"regexp"
	"strings"
)

// The wiki pages kamui knows about. These are file names inside
// https://github.com/fmhy/edit/tree/main/docs.
//
// Note: the original design doc referenced "videopiracyguide.md". That file no
// longer exists; FMHY split it into per-topic pages in 2024.
const (
	DocVideo   = "video.md"   // movies, TV, anime, live sport
	DocAudio   = "audio.md"   // music
	DocReading = "reading.md" // books, comics
)

// Source is one website listed in the wiki.
type Source struct {
	// Name is the link text, e.g. "ZetMoon".
	Name string
	// URL is the site's address, e.g. "https://zetmoon.live/".
	URL string
	// Path is the chain of section headings containing this source,
	// e.g. ["Streaming Sites", "Multi-Server"]. Ordered, outermost first.
	Path []string
	// Starred is true when FMHY marks the entry with a star. The wiki uses
	// stars for its own top picks, which is a free ranking signal for us.
	Starred bool
	// Order is the source's line order in the document. FMHY curates by
	// hand, so earlier usually means better. Order preserves that.
	Order int
}

// Section returns the innermost section name, or "" if the source sits
// directly under a top-level heading.
func (s Source) Section() string {
	if len(s.Path) == 0 {
		return ""
	}
	return s.Path[len(s.Path)-1]
}

// Host returns the site's domain with any "www." prefix removed, which is
// what we group and deduplicate on.
func (s Source) Host() string {
	u, err := url.Parse(s.URL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

var (
	// A markdown heading: one to six '#' characters, then the title.
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)

	// A bullet point: optional indent, then '*' or '-', then a space.
	// Requiring the space is what makes "**[bold**" not count as a bullet.
	bulletRe = regexp.MustCompile(`^\s*[*-]\s+`)

	// A markdown link: [label](url). Only http(s) is accepted.
	linkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+)\)`)

	// A link label that is nothing but digits, e.g. the "[2]" in a mirror list.
	numericLabelRe = regexp.MustCompile(`^\d+$`)
)

// headingFrame is one open heading on the section stack.
type headingFrame struct {
	level int
	title string
}

// noiseLabels are trailing links that annotate a site rather than name one.
// They must be dropped before we decide which link is the real entry.
var noiseLabels = map[string]bool{
	"status":   true,
	"note":     true,
	"discord":  true,
	"telegram": true,
	"github":   true,
	"mirror":   true,
	"mirrors":  true,
	"wiki":     true,
	"reddit":   true,
}

// Parse turns one FMHY markdown page into a flat list of sources, in document
// order.
//
// The tricky part is knowing where a section ends. Markdown has no closing
// tag: a section runs until the next heading of the same or a higher level.
// We track that with a small stack, which is also how we build Source.Path.
func Parse(markdown string) []Source {
	var (
		stack   []headingFrame
		sources []Source
	)

	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimRight(line, "\r")

		if m := headingRe.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			title := cleanHeading(m[2])

			// Close every section at this depth or deeper. A level-2 heading
			// therefore ends the previous level-2 section, and also any
			// level-3 subsections nested inside it.
			for len(stack) > 0 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, headingFrame{level: level, title: title})
			continue
		}

		if !bulletRe.MatchString(line) {
			continue
		}
		src, ok := parseBullet(line)
		if !ok {
			continue
		}
		src.Path = pathOf(stack)
		src.Order = len(sources)
		sources = append(sources, src)
	}

	return sources
}

// parseBullet extracts the primary source from a single bullet line.
//
// A bullet may hold several links, like:
//
//   - ⭐ **[1Shows](http://1shows.org)**, [1Flex](https://1flex.org/) or [1Tube](https://1tube.org/) - 8 Sources
//
// The first link is the site; the rest are alternates or mirrors, and any
// [Status]/[Discord] links at the end are metadata. So: take the first link
// that is neither a numeric mirror marker nor a known metadata label.
func parseBullet(line string) (Source, bool) {
	// Strip bold so "**[Name](url)**" becomes "[Name](url)" and the link
	// regex can see it.
	clean := strings.ReplaceAll(line, "**", "")

	for _, m := range linkRe.FindAllStringSubmatch(clean, -1) {
		label := strings.TrimSpace(m[1])
		if numericLabelRe.MatchString(label) || noiseLabels[strings.ToLower(label)] {
			continue
		}
		return Source{
			Name:    label,
			URL:     m[2],
			Starred: strings.Contains(line, "⭐"),
		}, true
	}

	return Source{}, false
}

// cleanHeading strips the wiki's decorative markers.
//
//	"► Streaming Sites"            -> "Streaming Sites"
//	"▷ [Educational](https://...)" -> "Educational"
func cleanHeading(title string) string {
	title = strings.TrimSpace(title)
	title = strings.TrimSpace(strings.TrimLeft(title, "►▷"))

	// Some headings are themselves links to another wiki page. Keep the label
	// so the tree stays readable, and drop the URL.
	if m := linkRe.FindStringSubmatch(title); m != nil {
		return strings.TrimSpace(m[1])
	}
	return title
}

func pathOf(stack []headingFrame) []string {
	out := make([]string, 0, len(stack))
	for _, f := range stack {
		out = append(out, f.title)
	}
	return out
}

// Dedupe removes repeated sites, keeping the first (highest-ranked) entry.
//
// The wiki lists the same site in several sections, and mirror domains repeat.
// Two entries count as the same site when their hosts match.
func Dedupe(sources []Source) []Source {
	seen := make(map[string]bool, len(sources))
	out := make([]Source, 0, len(sources))

	for _, s := range sources {
		key := s.Host()
		if key == "" {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out
}
