package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// Wordnik fetches today's Word of the Day from Wordnik's public API and
// emits "Word of the Day|WORD, pos. definition|" — shaped so the
// DictionaryScene's dictionaryEntryRE parses it the same way it parses
// Devil's Dictionary and Jargon File entries.
//
// When WORDNIK_API_KEY is unset (or a request fails) Wordnik falls back to
// a tiny day-of-year-indexed list so the scene always renders something
// instead of going blank. The fallback is intentionally short — its job
// is to keep the slot populated for the small fraction of operators
// without an API key, not to be a real curated wordlist.
type Wordnik struct {
	client *http.Client
	apiKey string
}

const wordnikEndpoint = "https://api.wordnik.com/v4/words.json/wordOfTheDay"

func NewWordnik() *Wordnik {
	return &Wordnik{
		client: &http.Client{Timeout: 30 * time.Second},
		apiKey: os.Getenv("WORDNIK_API_KEY"),
	}
}

func (w *Wordnik) Name() string { return "quotes/Wordnik" }

// wordnikResponse is the slice of Wordnik's WOTD JSON we actually use.
// The real response carries notes, examples, etymology, and more; the
// scene only renders headword + POS + first definition, so we ignore
// the rest.
type wordnikResponse struct {
	Word        string `json:"word"`
	Definitions []struct {
		Text         string `json:"text"`
		PartOfSpeech string `json:"partOfSpeech"`
	} `json:"definitions"`
	// Pronunciations is the Wordnik WOTD's optional pronunciation list.
	// We surface the first entry's Raw value (typically IPA) as the
	// fourth pipe segment so the ceremony-style scene can render it
	// beside the POS tag. RawType is ignored — the StyleCeremony layout
	// only displays the string, not its notation system.
	Pronunciations []struct {
		Raw     string `json:"raw"`
		RawType string `json:"rawType"`
	} `json:"pronunciations"`
}

func (w *Wordnik) Fetch(ctx context.Context) (string, error) {
	if w.apiKey == "" {
		return wordnikFallback(time.Now()), nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wordnikEndpoint+"?api_key="+w.apiKey, nil)
	if err != nil {
		return "", fmt.Errorf("wordnik: build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("wordnik: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wordnik: http %d", resp.StatusCode)
	}
	var body wordnikResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("wordnik: decode: %w", err)
	}
	if body.Word == "" || len(body.Definitions) == 0 {
		return "", fmt.Errorf("wordnik: empty word or definitions")
	}
	def := body.Definitions[0]
	if def.Text == "" {
		return "", fmt.Errorf("wordnik: empty first definition")
	}
	pron := ""
	if len(body.Pronunciations) > 0 {
		pron = strings.TrimSpace(body.Pronunciations[0].Raw)
	}
	return formatWOTD(body.Word, def.PartOfSpeech, def.Text, pron), nil
}

// formatWOTD builds the pipe-delimited line the DictionaryScene parser
// expects. POS is abbreviated to the atoms dictionaryEntryRE allows
// (n, v, adj, adv, prep, conj, interj, pron); unknown values are
// dropped so the regex's POS-optional fallback still extracts the
// headword. The definition has Wordnik's HTML tags (<xref>, <i>) stripped
// since the device renderer treats them as literal text. pron, when
// non-empty, is surfaced as a fourth pipe segment so StyleCeremony can
// render it beside the POS tag; the baked fallback list passes "".
func formatWOTD(word, pos, definition, pron string) string {
	entry := strings.ToUpper(word) + ", " + abbreviatePOS(pos) + " " + stripTags(definition)
	return "Word of the Day|" + entry + "||" + pron
}

// abbreviatePOS maps Wordnik's part-of-speech strings to the atoms the
// scene regex allows. Unknown values return "n." as a safe default —
// the headword is more important than perfect grammar labelling, and
// "n." is the most common POS for an English WOTD.
func abbreviatePOS(pos string) string {
	switch strings.ToLower(strings.TrimSpace(pos)) {
	case "noun":
		return "n."
	case "verb", "verb-transitive", "verb-intransitive":
		return "v."
	case "adjective":
		return "adj."
	case "adverb":
		return "adv."
	case "preposition":
		return "prep."
	case "conjunction":
		return "conj."
	case "interjection":
		return "interj."
	case "pronoun":
		return "pron."
	default:
		return "n."
	}
}

// stripTags removes Wordnik's inline XML markup (<xref>, <i>, <em>, etc.)
// without pulling in a real HTML parser. The device's Text renderer
// doesn't interpret HTML, so leaving the tags in would surface raw
// "<xref>" strings on the wall.
func stripTags(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// wordnikFallback picks a baked-in entry by day-of-year so the same
// word shows for the whole day even without an API key. The list is
// deliberately small — see the Wordnik type comment.
func wordnikFallback(now time.Time) string {
	w := wordnikFallbackList[now.YearDay()%len(wordnikFallbackList)]
	return "Word of the Day|" + w + "||"
}

// wordnikFallbackList is the no-key fallback. Each entry already matches
// the dictionaryEntryRE shape ("HEADWORD, pos. definition") so it flows
// through the scene the same way a live API response would.
var wordnikFallbackList = []string{
	"SERENDIPITY, n. The faculty of making fortunate discoveries by accident.",
	"PETRICHOR, n. The pleasant earthy smell after rain falls on dry soil.",
	"EPHEMERAL, adj. Lasting for a very short time.",
	"SONDER, n. The realization that each passerby has a life as vivid and complex as your own.",
	"DEFENESTRATE, v. To throw a person or thing out of a window.",
	"LIMINAL, adj. Occupying a position at, or on both sides of, a boundary or threshold.",
	"SUSURRUS, n. Whispering, murmuring, or rustling.",
	"MELLIFLUOUS, adj. Sweet or musical; pleasant to hear.",
	"PALIMPSEST, n. A manuscript on which the original writing has been effaced to make room for later writing.",
	"NEPENTHE, n. A drug or drink, or anything else, capable of making one forget grief or suffering.",
	"VELLICHOR, n. The strange wistfulness of used bookshops.",
	"APRICITY, n. The warmth of the sun in winter.",
	"INEFFABLE, adj. Too great or extreme to be expressed in words.",
	"NUMINOUS, adj. Having a strong religious or spiritual quality; suggesting the presence of a divinity.",
	"SOLIVAGANT, adj. Wandering alone.",
	"HIRAETH, n. A homesickness for a home to which you cannot return, a home which maybe never was.",
	"QUIDDITY, n. The inherent nature or essence of someone or something.",
	"CRYPTIC, adj. Having a meaning that is mysterious or obscure.",
	"AUTODIDACT, n. A self-taught person.",
	"GLOAMING, n. Twilight; dusk.",
	"WANDERLUST, n. A strong desire to travel.",
	"PETULANT, adj. Childishly sulky or bad-tempered.",
	"OBSTREPEROUS, adj. Noisy and difficult to control.",
	"SAUDADE, n. A deep emotional state of melancholic longing for a person or thing one loves.",
	"YUGEN, n. An awareness of the universe that triggers emotional responses too deep for words.",
	"CIRCUMLOCUTION, n. The use of many words where fewer would do, especially to be vague or evasive.",
	"PERSPICACIOUS, adj. Having a ready insight into and understanding of things.",
	"VICISSITUDE, n. A change of circumstances or fortune, typically one that is unwelcome.",
}

var _ widget.Widget = (*Wordnik)(nil)
