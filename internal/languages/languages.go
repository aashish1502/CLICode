// Package languages answers two questions the editor has to keep apart:
// which languages a user may write in, and what to put in the buffer when a
// problem ships no starter code for one of them.
//
// The supported set is a property of the judge, not of the problem. A judge that
// accepts Kotlin accepts it whether or not anyone wrote a Kotlin stub, so the
// set of languages a user can choose must never be derived from the set that
// happens to have starter code.
//
// Today the supported set comes from config, or a built-in default. Once the API
// exists it comes from the server's capabilities response, cached locally.
// Nothing above this package should be able to tell the difference.
package languages

import "context"

// Language describes one language a buffer can be opened in.
//
// ID is a plain string rather than a typed constant on purpose: the server must
// be able to start returning a language this client has never heard of without
// anyone shipping a new client. See Get.
type Language struct {
	ID            string // canonical id, as the judge knows it: "kotlin"
	DisplayName   string // "Kotlin"
	CommentPrefix string // "//", "#", "--"
}

// FallbackStub is the placeholder buffer for a language this problem ships no
// starter code for. It is a comment in that language, so the buffer is never
// syntactically invalid the moment it opens.
func (l Language) FallbackStub() string {
	prefix := l.CommentPrefix
	if prefix == "" {
		prefix = defaultCommentPrefix
	}
	return prefix + " Write your solution here\n"
}

// Catalog reports which languages a user may write in.
//
// The context and error are unused by the static implementation and exist for
// the API-backed one that replaces it, which will do I/O and can fail.
type Catalog interface {
	Supported(ctx context.Context) ([]Language, error)
}

const defaultCommentPrefix = "//"

// registry is what this client happens to know about. It is a convenience for
// presentation only — an id missing from here is still perfectly usable.
var registry = map[string]Language{
	"c":          {ID: "c", DisplayName: "C", CommentPrefix: "//"},
	"cpp":        {ID: "cpp", DisplayName: "C++", CommentPrefix: "//"},
	"csharp":     {ID: "csharp", DisplayName: "C#", CommentPrefix: "//"},
	"go":         {ID: "go", DisplayName: "Go", CommentPrefix: "//"},
	"java":       {ID: "java", DisplayName: "Java", CommentPrefix: "//"},
	"javascript": {ID: "javascript", DisplayName: "JavaScript", CommentPrefix: "//"},
	"kotlin":     {ID: "kotlin", DisplayName: "Kotlin", CommentPrefix: "//"},
	"php":        {ID: "php", DisplayName: "PHP", CommentPrefix: "//"},
	"python":     {ID: "python", DisplayName: "Python", CommentPrefix: "#"},
	"ruby":       {ID: "ruby", DisplayName: "Ruby", CommentPrefix: "#"},
	"rust":       {ID: "rust", DisplayName: "Rust", CommentPrefix: "//"},
	"scala":      {ID: "scala", DisplayName: "Scala", CommentPrefix: "//"},
	"swift":      {ID: "swift", DisplayName: "Swift", CommentPrefix: "//"},
	"typescript": {ID: "typescript", DisplayName: "TypeScript", CommentPrefix: "//"},
}

// Get returns what is known about a language id.
//
// An unrecognised id is not an error. It comes back usable, displayed under its
// own id and commented with "//", so a language the server adds tomorrow works
// in the client shipped today. That tolerance is the point of this function.
func Get(id string) Language {
	if l, ok := registry[id]; ok {
		return l
	}
	return Language{ID: id, DisplayName: id, CommentPrefix: defaultCommentPrefix}
}

// DefaultIDs is the supported set used when config names none. Kept deliberately
// small: it is a starting point for a local install, not a claim about what any
// particular judge accepts.
func DefaultIDs() []string {
	return []string{"python", "cpp", "java", "go"}
}

// Static is a Catalog backed by a fixed list of ids, from config or the default.
type Static struct {
	ids []string
}

func NewStatic(ids []string) Static {
	if len(ids) == 0 {
		ids = DefaultIDs()
	}
	return Static{ids: ids}
}

func (s Static) Supported(context.Context) ([]Language, error) {
	out := make([]Language, 0, len(s.ids))
	for _, id := range s.ids {
		out = append(out, Get(id))
	}
	return out, nil
}

// Union appends any ids not already present in supported, preserving the
// supported order. A problem that ships a stub for a language the catalog has
// not heard of is still reachable, rather than being silently dropped.
func Union(supported []Language, ids []string) []Language {
	seen := make(map[string]bool, len(supported))
	out := make([]Language, 0, len(supported)+len(ids))

	for _, l := range supported {
		seen[l.ID] = true
		out = append(out, l)
	}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, Get(id))
		}
	}
	return out
}

// IDs flattens a language list to its ids.
func IDs(langs []Language) []string {
	out := make([]string, 0, len(langs))
	for _, l := range langs {
		out = append(out, l.ID)
	}
	return out
}
