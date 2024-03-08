package rerules

import "github.com/tinne26/ggfnt"

const notNowGlyphFSM = "can't modify GlyphFSM while operating"
const uninitRunGlyphFSM = "invoked method without initial GlyphFSM.Begin()"

// --- FSM format description ---
// Target state ID (0 - 255) [with fsmStateOffsets], 
// then a series of target states. For each one, we have the ID,
// and a control byte. The control byte has 1 bit reserved for
// "last state", another reserved for "has ranges" (which means
// after this control byte we will have a control glyph with the
// number of ranges defined (0 - 255) and then N pairs of glyph
// indices or runes), and then in the remaining 6 bits we encode
// the number of single elements (glyphs or runes) that come next
// defining a potential transition. Each transition includes its
// condition as the first byte.

// TODO: tricky cases: when multiple rules overlap, I have to act greedily,
// but otherwise still have the "previous match" apply. I don't know if I replace
// these dynamically in tempAccGlyphs or if I need an additional structure to
// manage it properly. to be seen. I think direct replacing is fine and easy.

type GlyphFSM struct {
	rules []ggfnt.GlyphRewriteRule
	tempAccGlyphs []ggfnt.GlyphIndex
	fsm []byte // compiled data for the finite state machine
	fsmStateOffsets []uint16 // references fsm
	fsmCurrentStateOffset uint16 // references fsm
	state uint8 // 0b01 => bit flag for unsynced, 0b10 => sync flag for operating
}

func (self *GlyphFSM) NumRules() int {
	return len(self.rules)
}

func (self *GlyphFSM) QueueSize() int {
	return len(self.tempAccGlyphs)
}

func (self *GlyphFSM) Sync() {
	if self.state & fsmFlagUnsynced == 0 { return } // already synced
	self.recompile()
}

func (self *GlyphFSM) IsOperating() bool {
	return self.state & fsmFlagOperating != 0
}

// Adds a rewrite rule to the glyph finite state machine. The rule
// is not immediately compiled, compilation only happens if invoking
// [GlyphFSM.Sync]() manually or on the next [GlyphFSM.Begin]() process.
//
// Due to the finite state machine recompilation, adding and deleting
// rewrite rules is non-trivial and shouldn't be done on a per-frame
// basis.
func (self *GlyphFSM) AddRule(rule ggfnt.GlyphRewriteRule) {
	if self.state & fsmFlagOperating != 0 { panic(notNowGlyphFSM) }

	self.rules = append(self.rules, rule)
	self.state |= fsmFlagUnsynced
}

func (self *GlyphFSM) DeleteRule(rule ggfnt.GlyphRewriteRule) bool {
	if self.state & fsmFlagOperating != 0 { panic(notNowGlyphFSM) }

	for i, _ := range self.rules {
		if self.rules[i].Equals(rule) {
			numRules := len(self.rules)
			if i + 1 < numRules {
				copy(self.rules[i : ], self.rules[i + 1 : ])
			}
			self.rules = self.rules[ : numRules - 1]
			self.state |= fsmFlagUnsynced
			return true
		}
	}
	return false
}

func (self *GlyphFSM) Begin() {
	if self.state & fsmFlagOperating != 0 { panic("GlyphFSM process already started") }
	self.Sync()
	self.state = fsmFlagOperating
	self.tempAccGlyphs = self.tempAccGlyphs[ : 0]
	self.fsmCurrentStateOffset = 0
}

// Must be called manually when changing from glyph indices to runes
// or similar to force a flush of potentially accumulated glyph indices.
func (self *GlyphFSM) BreakSequence(each func(ggfnt.GlyphIndex)) {
	if self.state & fsmFlagOperating == 0 { panic(uninitRunGlyphFSM) }
	if self.state & fsmFlagUnsynced  != 0 { panic(brokenCode) }

	for _, glyphIndex := range self.tempAccGlyphs { each(glyphIndex) }
	self.tempAccGlyphs = self.tempAccGlyphs[ : 0]
	self.fsmCurrentStateOffset = 0
}

func (self *GlyphFSM) Finish(each func(ggfnt.GlyphIndex)) {
	self.BreakSequence(each)
	self.state = 0b00
}

// Notice: control glyph indices must be checked / cleared before invoking
// this function. Any control glyph will make Feed fail.
func (self *GlyphFSM) Feed(glyphIndex ggfnt.GlyphIndex, each func(ggfnt.GlyphIndex)) {
	if self.state & fsmFlagOperating == 0 { panic(uninitRunGlyphFSM) }
	if glyphIndex == ggfnt.GlyphZilch { return } // ignore zilch glyphs
	ensureValidFsmGlyphIndex(glyphIndex)

	panic("unimplemented")
}

func (self *GlyphFSM) recompile() {
	if self.state & fsmFlagOperating != 0 { panic(notNowGlyphFSM) }
	self.state = fsmFlagUnsynced | fsmFlagOperating
	// ... (magic)
	self.state = 0b00
}
