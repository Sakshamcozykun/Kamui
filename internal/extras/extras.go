// Package extras holds a small, hand-curated catalogue of sources that are
// not in the FMHY wiki.
//
// FMHY is the primary list, but it is not the only community directory worth
// reading. These entries were collected from:
//
//   - EverythingMoe (https://everythingmoe.com/) — ranked anime, manga,
//     manhwa and novel directories
//   - awesome-anime-sources (https://github.com/anshumanv/awesome-anime-sources)
//   - Awesome Piracy (https://github.com/Igglybuff/awesome-piracy)
//   - Awesome Entertainment (https://github.com/OshekharO/awesome-entertainment)
//
// Only sites that answered a normal browser-style request when this list was
// assembled are included. Sites that are JavaScript-only still appear here:
// kamui's generic search will quietly return nothing from them, but their
// link is still useful to open in a browser. The reachability probe decides
// at run time which ones are actually usable from the current network.
package extras

import (
	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

// Root is the first path element of every extra source. It keeps these
// entries visually separate from the wiki in "kamui sources" and lets the
// category commands tell them apart when applying --max.
const Root = "Other Lists"

type entry struct {
	name    string
	url     string
	starred bool
}

type category struct {
	section string
	entries []entry
}

// The paths are chosen to match the section selectors in cmd/common.go:
// "anime streaming", "manga", "light novels" and "ebook".
var categories = []category{
	{
		section: "Streaming Sites",
		entries: []entry{
			{"AZMovies", "https://azmovies.xyz/", false},
			{"Atlantic", "https://atlantic.st/", false},
			{"FlyStream", "https://flystream.net/", false},
			{"Levidia", "https://www.levidia.ch/", false},
			{"Movie123", "http://movie123.club/", false},
			{"Sereal", "https://www.sereal.com", false},
			{"Streamlord", "http://www.streamlord.com/", false},
			{"Watchroo", "https://www.watchroo.com/", false},
		},
	},
	{
		section: "Anime Streaming",
		entries: []entry{
			{"9anime", "https://www2.9anime.to/", false},
			{"AniSnatch", "https://anisnatch.to/home", true},
			{"Anime Cube", "https://animecube.live/", true},
			{"Anime-Loads", "https://www.anime-loads.org/", false},
			{"AniDub", "https://online.anidub.com/", false},
			{"AniLibria", "https://anilibria.tv/", false},
			{"Animelon", "https://animelon.com/", false},
			{"AnimeShow", "https://animeshow.tv", false},
			{"AnimeSub", "https://animesub.lt", false},
			{"AnimeVibe", "https://animevibe.ru/", false},
			{"AnimeVost", "https://animevost.org/", false},
			{"AniStar", "https://anistar.org", false},
			{"Otaku-Streamers", "https://otaku-streamers.com/", false},
		},
	},
	{
		section: "Manga",
		entries: []entry{
			{"Bato", "https://bato.to/", true},
			{"FanFox", "https://fanfox.net/", false},
			{"LikeManga", "https://likemanga.ink/", false},
			{"MangaCloud", "https://mangacloud.org/", false},
			{"MangaDex", "https://mangadex.org/", true},
			{"MangaFire", "https://mangafire.to/", true},
			{"MangaHere", "https://mangahere.cc/", false},
			{"MangaKakalot", "https://www.mangakakalot.gg/", false},
			{"MangaPark", "https://mangapark.net/", false},
			{"MangaPill", "https://mangapill.com/", false},
			{"Mangaball", "https://mangaball.com/", false},
			{"Mangago", "https://www.mangago.me/", false},
			{"Manhwatop", "https://manhwatop.com/", false},
			{"ReadManga", "https://www.readmng.com/", false},
			{"ReadManhua", "https://readmanhua.net/", false},
			{"Swarm", "https://swarm.ws/", false},
			{"Toonz", "https://toonz.to/", false},
			{"Webnex", "https://webnex.cc/", false},
			{"WebtoonClub", "https://webtoonclub.com/", false},
			{"Webtoons", "https://www.webtoons.com/en/", true},
			{"XComic", "https://xcomic.me/", false},
		},
	},
	{
		section: "Light Novels",
		entries: []entry{
			{"Bunkobon", "https://bunkobon.moe/", false},
			{"FANMTL", "https://www.fanmtl.com/", false},
			{"Fictionzone", "https://fictionzone.net/", false},
			{"LNMTL", "https://lnmtl.com/", false},
			{"LnCrawler", "https://lncrawler.monster/", false},
			{"LightNovel.to", "https://lightnovel.to/", false},
			{"MTLBooks", "https://mtlbooks.com/", false},
			{"MVLEMPYR", "https://www.mvlempyr.com/", false},
			{"Novel Ping", "https://novelping.com/", false},
			{"NovelArchive", "https://novelarchive.st/", false},
			{"NovelBuddy", "https://novelbuddy.com/", false},
			{"NovelDex", "https://noveldex.io/", false},
			{"Novelhall", "https://www.novelhall.com/", false},
			{"Novelito", "https://noveli.to/", false},
			{"TheNovel", "https://thenovel.net/", false},
			{"WTR-Lab", "https://wtr-lab.com/", true},
			{"Wuxia", "https://wuxia.click/", false},
			{"Xnovel", "https://xnovel.me/", false},
		},
	},
	{
		section: "Ebooks",
		entries: []entry{
			{"All IT eBooks", "http://www.allitebooks.com/", false},
			{"ForCoder", "https://forcoder.su/", false},
			{"IT eBooks", "http://it-ebooks.info/", false},
			{"MagazineLib", "https://magazinelib.com/", false},
			{"ManyBooks", "https://manybooks.net/", false},
			{"Memory of the World", "http://library.memoryoftheworld.org/", false},
			{"PDF Drive", "https://www.pdfdrive.com/", true},
			{"PDF Giant", "http://pdf-giant.com/", false},
			{"Standard Ebooks", "https://standardebooks.org/", true},
		},
	},
}

// Sources returns every extra as a catalogue source. Entries are assigned
// sequential order numbers so they keep their curated order.
func Sources() []fmhy.Source {
	out := make([]fmhy.Source, 0, 96)
	order := 1
	for _, c := range categories {
		for _, e := range c.entries {
			out = append(out, fmhy.Source{
				Name:    e.name,
				URL:     e.url,
				Path:    []string{Root, c.section},
				Starred: e.starred,
				Order:   order,
			})
			order++
		}
	}
	return out
}

// IsExtra reports whether a source came from this package rather than FMHY.
func IsExtra(s fmhy.Source) bool {
	return len(s.Path) > 0 && s.Path[0] == Root
}
