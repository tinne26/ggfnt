package rerules

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal/rerules/utf8rule"

type Utf8Tester struct {
	tester utf8rule.Tester
}

// --- general operations ---

func (self *Utf8Tester) NumRules() int {
	return self.tester.NumRules()
}

func (self *Utf8Tester) IsOperating() bool {
	return self.tester.IsOperating()
}

func (self *Utf8Tester) NumPendingRunes() int {
	return self.tester.NumPendingRunes()
}

func (self *Utf8Tester) Resync(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) error {
	return self.tester.Resync(font, settingsCache)
}

// --- rule management ---

func (self *Utf8Tester) RemoveAllRules() {
	self.tester.RemoveAllRules()
}

func (self *Utf8Tester) AddRule(rule ggfnt.Utf8RewriteRule) error {
	return self.tester.AddRule(rule)
}

func (self *Utf8Tester) RemoveRule(rule ggfnt.Utf8RewriteRule) bool {
	return self.tester.RemoveRule(rule)
}

// --- condition control ---

func (self *Utf8Tester) RefreshConditions(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) {
	self.tester.RefreshConditions(font, settingsCache)
}

// --- rune sequence operations ---

func (self *Utf8Tester) BeginSequence(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) error {
	return self.tester.BeginSequence(font, settingsCache)
}

func (self *Utf8Tester) Feed(codePoint rune, fn func(rune)) error {
	return self.tester.Feed(codePoint, fn)
}

func (self *Utf8Tester) Break(fn func(rune)) {
	self.tester.Break(fn)
}

func (self *Utf8Tester) FinishSequence(fn func(rune)) {
	self.tester.FinishSequence(fn)
}
