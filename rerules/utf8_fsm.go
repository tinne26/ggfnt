package rerules

import "github.com/tinne26/ggfnt"

const notNowUtf8FSM = "can't modify Utf8FSM while operating"
const uninitRunUtf8FSM = "invoked method without initial Utf8FSM.Begin()"

type Utf8FSM struct {
	rules []ggfnt.Utf8RewriteRule
	tempAccRunes []rune
	fsm []byte
	fsmStateOffsets []uint16 // references fsm
	fsmCurrentStateOffset uint16 // references fsm
	state uint8 // 0b01 => bit flag for unsynced, 0b10 => sync flag for operating
}

func (self *Utf8FSM) NumRules() int {
	return len(self.rules)
}

func (self *Utf8FSM) QueueSize() int {
	return len(self.tempAccRunes)
}

func (self *Utf8FSM) Sync() {
	if self.state & fsmFlagUnsynced == 0 { return } // already synced
	self.recompile()
}

func (self *Utf8FSM) IsOperating() bool {
	return self.state & fsmFlagOperating != 0
}

func (self *Utf8FSM) AddRule(rule ggfnt.Utf8RewriteRule) {
	if self.state & fsmFlagOperating != 0 { panic(notNowUtf8FSM) }

	self.rules = append(self.rules, rule)
	self.state |= fsmFlagUnsynced
}

func (self *Utf8FSM) DeleteRule(rule ggfnt.Utf8RewriteRule) bool {
	if self.state & fsmFlagOperating != 0 { panic(notNowUtf8FSM) }

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

func (self *Utf8FSM) Begin() {
	if self.state & fsmFlagOperating != 0 { panic("Utf8FSM process already started") }
	self.Sync()
	self.state = fsmFlagOperating
	self.tempAccRunes = self.tempAccRunes[ : 0]
	self.fsmCurrentStateOffset = 0
}

// Must be called manually when changing from glyph indices to runes
// or similar to force a flush of potentially accumulated glyph indices.
func (self *Utf8FSM) BreakSequence(each func(rune)) {
	if self.state & fsmFlagOperating == 0 { panic(uninitRunGlyphFSM) }
	if self.state & fsmFlagUnsynced  != 0 { panic(brokenCode) }

	for _, codePoint := range self.tempAccRunes { each(codePoint) }
	self.tempAccRunes = self.tempAccRunes[ : 0]
	self.fsmCurrentStateOffset = 0
}

func (self *Utf8FSM) Finish(each func(rune)) {
	self.BreakSequence(each)
	self.state = 0b00
}

// ...
func (self *Utf8FSM) Feed(codePoint rune, eachCodePoint func(rune)) {
	// ...
}


func (self *Utf8FSM) recompile() {
	if self.state & fsmFlagOperating != 0 { panic(notNowUtf8FSM) }
	self.state = fsmFlagUnsynced | fsmFlagOperating
	// ... (magic)
	self.state = 0b00
}
