package rerules

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal/rerules/glyphrule"

type GlyphTester struct {
	tester glyphrule.Tester
}

// --- general operations ---

func (self *GlyphTester) NumRules() int {
	return self.tester.NumRules()
}

func (self *GlyphTester) IsOperating() bool {
	return self.tester.IsOperating()
}

func (self *GlyphTester) NumPendingGlyphs() int {
	return self.tester.NumPendingGlyphs()
}

func (self *GlyphTester) Resync(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) error {
	return self.tester.Resync(font, settingsCache)
}

// --- rule management ---

func (self *GlyphTester) RemoveAllRules() {
	self.tester.RemoveAllRules()
}

func (self *GlyphTester) AddRule(rule ggfnt.GlyphRewriteRule) error {
	return self.tester.AddRule(rule)
}

func (self *GlyphTester) RemoveRule(rule ggfnt.GlyphRewriteRule) bool {
	return self.tester.RemoveRule(rule)
}

// --- condition control ---

func (self *GlyphTester) RefreshConditions(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) {
	self.tester.RefreshConditions(font, settingsCache)
}

// --- glyph sequence operations ---

func (self *GlyphTester) BeginSequence(font *ggfnt.Font, settingsCache *ggfnt.SettingsCache) error {
	return self.tester.BeginSequence(font, settingsCache)
}

func (self *GlyphTester) Feed(glyphIndex ggfnt.GlyphIndex, fn func(ggfnt.GlyphIndex)) error {
	return self.tester.Feed(glyphIndex, fn)
}

func (self *GlyphTester) Break(fn func(ggfnt.GlyphIndex)) {
	self.tester.Break(fn)
}

func (self *GlyphTester) FinishSequence(fn func(ggfnt.GlyphIndex)) {
	self.tester.FinishSequence(fn)
}
