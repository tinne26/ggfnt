package ggfnt

// TODO:
// - say we have a setting for filled/unfilled glyphs. maybe we want to use that
//   dynamically as an animation form instead. it would be great to allow *not
//   caching* certain settings and switches, intentionally, to allow fast setting
//   changes. like, "MarkAsLiveSetting(setting uint8, live bool)". or for a 
//   transition between filled and unfilled, which would be cool.

type SettingsCache struct {
	mappingSettingRelevanceFlags []uint8 // compressed bool slice
	rewriteConditionsSettingRelevanceFlags []uint8 // compressed bool slice
	mappingCaseCachingFlags []uint8 // compressed bool slice
	rewriteConditionsCachingFlags []uint8 // compressed bool slice
	mappingCachedCases []uint8
	rewriteConditionsCachedBools []uint8 // compressed bool slice
	settings []uint8 // reference to current settings
}

func NewSettingsCache(font *Font) *SettingsCache {
	var cache SettingsCache
	
	numSettings := font.Settings().Count()
	if numSettings > 0 {
		requiredBytes := (int(numSettings) + 7) >> 3
		cache.mappingSettingRelevanceFlags = make([]uint8, requiredBytes)
		cache.rewriteConditionsSettingRelevanceFlags = make([]uint8, requiredBytes)
	}
	
	numSwitchTypes := font.Mapping().NumSwitchTypes()
	if numSwitchTypes > 0 {
		cache.mappingCaseCachingFlags = make([]uint8, (int(numSwitchTypes) + 7) >> 3)
		cache.mappingCachedCases = make([]uint8, int(numSwitchTypes))
	}

	numConditions := font.Rewrites().NumConditions()
	if numConditions > 0 {
		requiredBytes := (int(numConditions) + 7 >> 3)
		cache.rewriteConditionsCachingFlags = make([]uint8, requiredBytes)
		cache.rewriteConditionsCachedBools = make([]uint8, requiredBytes)
	}

	// TODO: gotta scan a lot of stuff to determine relevance flags

	return &cache
}

// TODO: read only, might remove later.
func (self *SettingsCache) UnsafeSlice() []uint8 {
	return self.settings
}

func (self *SettingsCache) Get(key SettingKey) uint8 {
	if len(self.settings) <= int(key) { return 0 }
	return self.settings[key]
}

// Returns two bools for mappings and rewrite conditions affected.
func (self *SettingsCache) Set(key SettingKey, option uint8) (mappingsAffected, rewriteConditionsAffected bool) {
	self.settings[key] = option
	wordIndex, bit := self.wordAndBit(uint8(key))
	
	// if setting is used for at least one switch/condition, drop cached values
	flags := self.mappingSettingRelevanceFlags[wordIndex]
	if flags & bit != 0 {
		clear(self.mappingCaseCachingFlags) // this is fast. 255 switches are 32 bytes, 4 64bit words
		mappingsAffected = true
	}
	flags  = self.rewriteConditionsSettingRelevanceFlags[wordIndex]
	if flags & bit != 0 {
		clear(self.rewriteConditionsCachingFlags)
		rewriteConditionsAffected = true
	}

	return mappingsAffected, rewriteConditionsAffected
}

// First uint8 is the case value, second bool is case being cached or
// not. If not cached, the first result must be ignored.
func (self *SettingsCache) GetMappingCase(switchTypeIndex uint8) (uint8, bool) {
	if switchTypeIndex == 255 { return 0, true } // default case
	wordIndex, bit := self.wordAndBit(switchTypeIndex)
	if self.mappingCaseCachingFlags[wordIndex] & bit == 0 { return 0, false }
	return self.mappingCachedCases[switchTypeIndex], true
}

func (self *SettingsCache) CacheMappingCase(switchTypeIndex uint8, result uint8) {
	wordIndex, bit := self.wordAndBit(switchTypeIndex)
	self.mappingCaseCachingFlags[wordIndex] |= bit
	self.mappingCachedCases[switchTypeIndex] = result
}

// First bool is the condition being satisfied or not. Second bool is
// condition being cached or not. If not cached, the first result must
// be ignored.
func (self *SettingsCache) GetRewriteCondition(conditionIndex uint8) (bool, bool) {
	if conditionIndex == 255 { return true, true } // default case
	wordIndex, bit := self.wordAndBit(conditionIndex)
	if self.rewriteConditionsCachingFlags[wordIndex] & bit == 0 { return false, false }
	return (self.rewriteConditionsCachedBools[wordIndex] & bit) != 0, true
}

func (self *SettingsCache) CacheRewriteCondition(conditionIndex uint8, satisfied bool) {
	wordIndex, bit := self.wordAndBit(conditionIndex)
	self.rewriteConditionsCachingFlags[wordIndex] |= bit
	if satisfied {
		self.rewriteConditionsCachedBools[wordIndex] |= bit
	} else {
		self.rewriteConditionsCachedBools[wordIndex] &= ^bit
	}
}

func (self *SettingsCache) wordAndBit(index uint8) (uint8, uint8) {
	return index >> 3, index & 0b111
}
