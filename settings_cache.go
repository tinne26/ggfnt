package ggfnt

import "github.com/tinne26/ggfnt/internal"

// TODO:
// - say we have a setting for filled/unfilled glyphs. maybe we want to use that
//   dynamically as an animation form instead. it would be great to allow *not
//   caching* certain settings and switches, intentionally, to allow fast setting
//   changes. like, "MarkAsLiveSetting(setting uint8, live bool)". or for a 
//   transition between filled and unfilled, which would be cool.

type SettingsCache struct {
	// relevance flags indicate which settings can affect 
	// regular glyph mapping and rewrite rules
	mappingSettingRelevanceFlags internal.BoolList
	rewriteConditionsSettingRelevanceFlags internal.BoolList

	// caching flags
	mappingCaseCachingFlags internal.BoolList
	rewriteConditionsCachingFlags internal.BoolList

	// cached values
	mappingCachedCases []uint8
	rewriteConditionsCachedBools internal.BoolList

	// actual settings
	settings []uint8 // reference to current settings
}

func NewSettingsCache(font *Font) *SettingsCache {
	var cache SettingsCache
	
	numSettings := int(font.Settings().Count())
	cache.settings = make([]uint8, numSettings)
	cache.mappingSettingRelevanceFlags = internal.NewBoolList(numSettings)
	cache.rewriteConditionsSettingRelevanceFlags = internal.NewBoolList(numSettings)
	cache.mappingSettingRelevanceFlags.SetAllTrue()
	cache.rewriteConditionsSettingRelevanceFlags.SetAllTrue()
	
	numSwitchTypes := int(font.Mapping().NumSwitchTypes())
	cache.mappingCaseCachingFlags = internal.NewBoolList(numSwitchTypes)
	cache.mappingCachedCases = make([]uint8, numSwitchTypes)

	numConditions := int(font.Rewrites().NumConditions())
	cache.rewriteConditionsCachingFlags = internal.NewBoolList(numConditions)
	cache.rewriteConditionsCachedBools = internal.NewBoolList(numConditions)

	// TODO: need to see if I can set relevance flags for rewrite rules and
	//       mapping conditions more precisely in a reasonable way than just
	//       SetAllTrue().

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
	
	// if setting is used for at least one switch/condition, drop cached values
	if self.mappingSettingRelevanceFlags.Get(int(key)) {
		self.mappingCaseCachingFlags.SetAllFalse()
		mappingsAffected = true
	}
	if self.rewriteConditionsSettingRelevanceFlags.Get(int(key)) {
		self.rewriteConditionsCachingFlags.SetAllFalse()
		rewriteConditionsAffected = true
	}

	return mappingsAffected, rewriteConditionsAffected
}

// First uint8 is the case value, second bool is case being cached or
// not. If not cached, the first result must be ignored.
func (self *SettingsCache) GetMappingCase(switchTypeIndex uint8) (uint8, bool) {
	if switchTypeIndex >= 254 { return 0, true } // default cases
	if self.mappingCaseCachingFlags.IsUnsetU8(switchTypeIndex) { return 0, false }
	return self.mappingCachedCases[switchTypeIndex], true
}

func (self *SettingsCache) CacheMappingCase(switchTypeIndex uint8, result uint8) {
	self.mappingCaseCachingFlags.SetU8(switchTypeIndex, true)
	self.mappingCachedCases[switchTypeIndex] = result
}

// First bool is the condition being satisfied or not. Second bool is
// condition being cached or not. If not cached, the first result must
// be ignored.
func (self *SettingsCache) GetRewriteCondition(conditionIndex uint8) (bool, bool) {
	if conditionIndex == 255 { return true, true } // default case
	if self.rewriteConditionsCachingFlags.IsUnsetU8(conditionIndex) { return false, false }
	return self.rewriteConditionsCachedBools.IsSetU8(conditionIndex), true
}

func (self *SettingsCache) CacheRewriteCondition(conditionIndex uint8, satisfied bool) {
	self.rewriteConditionsCachingFlags.SetU8(conditionIndex, true)
	self.rewriteConditionsCachedBools.SetU8(conditionIndex, satisfied)
}
