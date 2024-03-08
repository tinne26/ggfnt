package ggfnt

type SwitchCache struct {
	settingRelevanceFlags []uint8 // compressed bool slice
	caseCachingFlags []uint8
	cachedCases []uint8
	settings []uint8
}

func NewSwitchCache(font *Font) *SwitchCache {
	var cache SwitchCache
	
	numSettings := font.Settings().Count()
	if numSettings > 0 {
		cache.settingRelevanceFlags = make([]uint8, (int(numSettings) + 7) >> 3)
	}
	
	numSwitchTypes := font.Mapping().NumSwitchTypes()
	if numSwitchTypes > 0 {
		cache.caseCachingFlags = make([]uint8, (int(numSwitchTypes) + 7) >> 3)
		cache.cachedCases = make([]uint8, int(numSwitchTypes))
	}

	return &cache
}

func (self *SwitchCache) GetSetting(key SettingKey) uint8 {
	if len(self.settings) <= int(key) { return 0 }
	return self.settings[key]
}

func (self *SwitchCache) GetCase(switchTypeIndex uint8) (uint8, bool) {
	if switchTypeIndex == 255 { return 0, true } // default case
	wordIndex, bit := self.wordAndBit(switchTypeIndex)
	if self.caseCachingFlags[wordIndex] & bit == 0 { return 0, false }
	return self.cachedCases[switchTypeIndex], true
}

func (self *SwitchCache) CacheCase(switchTypeIndex uint8, result uint8) {
	wordIndex, bit := self.wordAndBit(switchTypeIndex)
	self.caseCachingFlags[wordIndex] |= bit
	self.cachedCases[switchTypeIndex] = result
}

func (self *SwitchCache) NotifySettingChange(settingKey SettingKey, settings []uint8) {
	wordIndex, bit := self.wordAndBit(uint8(settingKey))
	flags := self.settingRelevanceFlags[wordIndex]
	if flags & bit == 0 { return } // setting not used for switches, ignore the drop

	// setting used for at least one switch. drop them all for speed.
	clear(self.caseCachingFlags) // this is fast enough. 255 switches would be 32 bytes, 4 64bit words
	self.settings = settings
}

func (self *SwitchCache) wordAndBit(index uint8) (uint8, uint8) {
	return index >> 3, index & 0b111
}
