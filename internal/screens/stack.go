package screens

import tea "github.com/charmbracelet/bubbletea"

// cacheSize is how many popped screens are kept alive so returning to one
// restores its state — an editor buffer, a scroll position — instead of
// rebuilding it from scratch. Small on purpose: it is a convenience, not
// storage. Once solutions live in SQLite the buffer survives without it.
const cacheSize = 8

// Key identifies a screen *instance*.
//
// Arg distinguishes instances of the same screen holding different content: the
// problem screen for problem 1 is not the problem screen for problem 2, so they
// get separate entries and separate cached state. An empty Arg means the screen
// is a singleton — there is only ever one menu.
type Key struct {
	ID  screenID
	Arg string
}

// Mode is how a push interacts with what is already on the stack.
type Mode int

const (
	// Single reuses the instance if an equal key is already on the stack,
	// popping back to it rather than pushing a duplicate.
	//
	// This is what makes a held-down navigation key harmless: the stack can
	// never grow past the number of distinct screens.
	Single Mode = iota

	// Root clears the stack; the pushed screen becomes its only entry.
	Root

	// Replace swaps the current entry, leaving the depth unchanged. Used where
	// going back to the screen being left would make no sense — the splash.
	Replace

	// Fresh always pushes a new instance, even if an equal key is present, and
	// never restores from the cache. Nothing uses it yet; it is the escape
	// hatch for a screen that genuinely needs to be re-entrant.
	Fresh
)

type frame struct {
	key   Key
	model tea.Model
}

// stack is the screen history. The last frame is what the user sees.
type stack struct {
	frames []frame
	cache  []frame // most-recently-used first, capped at cacheSize
}

func (s *stack) depth() int { return len(s.frames) }

func (s *stack) current() (tea.Model, bool) {
	if len(s.frames) == 0 {
		return nil, false
	}
	return s.frames[len(s.frames)-1].model, true
}

// replaceCurrent swaps the model of the top frame, keeping its key. Bubble Tea
// models are values, so every Update produces a new one that has to be stored.
func (s *stack) replaceCurrent(m tea.Model) {
	if len(s.frames) == 0 {
		return
	}
	s.frames[len(s.frames)-1].model = m
}

// reset discards everything and starts over with a single frame.
func (s *stack) reset(k Key, m tea.Model) {
	s.frames = []frame{{key: k, model: m}}
	s.cache = nil
}

// push places a screen on top, honouring mode. The returned command is the new
// screen's Init — nil when an existing or cached instance was reused, since
// re-running Init would restart timers that are already running.
func (s *stack) push(k Key, mode Mode, build func() tea.Model) tea.Cmd {
	switch mode {
	case Root:
		s.truncate(0)

	case Replace:
		s.truncate(len(s.frames) - 1)

	case Single:
		if i := s.indexOf(k); i >= 0 {
			// Already live: discard everything above it and reveal it.
			s.truncate(i + 1)
			return nil
		}
	}

	if mode != Fresh {
		if m, ok := s.uncache(k); ok {
			s.frames = append(s.frames, frame{key: k, model: m})
			return nil
		}
	}

	m := build()
	s.frames = append(s.frames, frame{key: k, model: m})
	return m.Init()
}

// pop removes the top screen and caches it. ok is false once the stack is
// empty, which is the signal to quit.
func (s *stack) pop() (tea.Cmd, bool) {
	if len(s.frames) == 0 {
		return nil, false
	}
	s.truncate(len(s.frames) - 1)

	if len(s.frames) == 0 {
		return nil, false
	}
	return nil, true
}

// broadcast delivers a message to every screen on the stack, not just the
// visible one, so a screen resized while hidden renders correctly on return.
func (s *stack) broadcast(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(s.frames))
	for i, f := range s.frames {
		updated, cmd := f.model.Update(msg)
		s.frames[i].model = updated
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (s *stack) indexOf(k Key) int {
	for i, f := range s.frames {
		if f.key == k {
			return i
		}
	}
	return -1
}

// truncate drops every frame from n upward, caching each so returning to one
// restores its state.
func (s *stack) truncate(n int) {
	if n < 0 {
		n = 0
	}
	for i := len(s.frames) - 1; i >= n; i-- {
		s.recache(s.frames[i])
	}
	s.frames = s.frames[:n]
}

// recache moves a frame to the front of the LRU, evicting the oldest past the cap.
func (s *stack) recache(f frame) {
	for i, c := range s.cache {
		if c.key == f.key {
			s.cache = append(s.cache[:i], s.cache[i+1:]...)
			break
		}
	}
	s.cache = append([]frame{f}, s.cache...)
	if len(s.cache) > cacheSize {
		s.cache = s.cache[:cacheSize]
	}
}

// uncache takes a screen back out of the cache, if it is still there.
func (s *stack) uncache(k Key) (tea.Model, bool) {
	for i, c := range s.cache {
		if c.key == k {
			s.cache = append(s.cache[:i], s.cache[i+1:]...)
			return c.model, true
		}
	}
	return nil, false
}
