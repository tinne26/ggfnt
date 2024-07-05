package ggfnt

// Notice: there are multiple strategies to caching, and one
// could avoid caching directly mapped glyphs, or cache even
// code point misses... this is just one reasonable implementation.

const noEntry uint16 = 65535

type mappingEntry struct {
	dropSignature uint32
	prevEntry uint16
	nextEntry uint16
	codePoint rune
	
	// direct values for the GlyphMappingGroup
	mappingOffset uint32
	caseBranch uint8
	directMapping bool
	switchType uint8
}

func (self *mappingEntry) ToMappingGroup(font *Font) GlyphMappingGroup {
	return GlyphMappingGroup{
		font: font,
		offset: self.mappingOffset,
		caseBranch: self.caseBranch,
		directMapping: self.directMapping,
	}
}

func (self *mappingEntry) Update(dropCounter uint32, group GlyphMappingGroup) {
	self.dropSignature = dropCounter
	self.mappingOffset = group.offset
	self.switchType    = group.switchType
	self.caseBranch    = group.caseBranch
	self.directMapping = group.directMapping
}

// Glyph mapping cache. These are recommended to be used at small sizes,
// like 192. In general, a glyph mapping cache is not even critical for
// operation unless the font has multiple settings affecting mapping,
// animations and so on. In those cases, the mapping can get more expensive,
// and using a mapping cache can be beneficial.
type MappingCache struct {
	font *Font
	dropCounter uint32
	mruIndex uint16
	lruIndex uint16
	cachedMappings map[rune]uint16
	mappingEntries []mappingEntry
}

// The size is statically allocated.
func NewMappingCache(font *Font, size int) *MappingCache {
	// safety assertions
	if size <= 0 { panic("mapping cache size must be positive") }
	if size > 65000 { panic("mapping cache size can't exceed 65k") }
	if font == nil { panic("mapping cache can't accept nil font") }

	cache := &MappingCache{
		font: font, 
		mruIndex: noEntry,
		lruIndex: 0,
		cachedMappings: make(map[rune]uint16, size),
		mappingEntries: make([]mappingEntry, size),
	}
	cache.initEntriesLinking()
	return cache
}

// Resets the mapping cache completely, with only the size being preserved.
func (self *MappingCache) Reset(font *Font) {
	self.font = font
	self.mruIndex = noEntry
	self.lruIndex = 0
	clear(self.mappingEntries)
	clear(self.cachedMappings)
	self.initEntriesLinking()
}

func (self *MappingCache) initEntriesLinking() {
	size := uint16(len(self.mappingEntries))
	for i := uint16(0); i < size; i++ {
		self.mappingEntries[i].prevEntry = i - 1
		self.mappingEntries[i].nextEntry = i + 1
	}
	self.mappingEntries[0].prevEntry = noEntry
	self.mappingEntries[size - 1].nextEntry = noEntry
}

// Drops must be manually requested due to variable changes.
func (self *MappingCache) Drop() {
	// technically overflows are possible, but it's not very realistic 
	// and wouldn't even break anything in practical cases
	self.dropCounter += 1
}

// Returned bool will be false if not found.
func (self *MappingCache) Get(codePoint rune, settings *SettingsCache) (GlyphMappingGroup, bool) {
	entryIndex, found := self.cachedMappings[codePoint]
	if found { // easy case
		entry := &self.mappingEntries[entryIndex]
		self.updateMRU(entry, entryIndex) // this happens in both incoming cases
		if entry.dropSignature == self.dropCounter {
			return entry.ToMappingGroup(self.font), true
		} else {
			caseBranch, cached := settings.GetMappingCase(entry.switchType)
			if cached && caseBranch == entry.caseBranch {
				// quick signature update, nothing really changed
				entry.dropSignature = self.dropCounter
				return entry.ToMappingGroup(self.font), true
			} else {
				// true mismatch, find new data and replace this entry
				glyphMapping, found := self.font.Mapping().Utf8WithCache(codePoint, settings)
				if !found { panic(brokenCode) } // can't have mappings cached if mappings don't exist
				entry.Update(self.dropCounter, glyphMapping)
				return glyphMapping, true
			}
		}
	} else { // replace LRU
		// find relevant data
		glyphMapping, found := self.font.Mapping().Utf8WithCache(codePoint, settings)
		if !found { return glyphMapping, false } // missing code point
		evictedCodePoint := self.mappingEntries[self.lruIndex].codePoint
		entry := &self.mappingEntries[self.lruIndex]
		entry.Update(self.dropCounter, glyphMapping)
		entry.codePoint = codePoint // code point also updated in this case

		// delete previous entry, link new one
		delete(self.cachedMappings, evictedCodePoint)
		self.cachedMappings[codePoint] = self.lruIndex

		// update MRU and return
		self.updateMRU(entry, self.lruIndex)
		return glyphMapping, found
	}
}

func (self *MappingCache) updateMRU(entry *mappingEntry, index uint16) {
	prevNext := entry.nextEntry
	if prevNext == noEntry { return } // already MRU
	
	prevPrev := entry.prevEntry
	if prevPrev == noEntry { // LRU case
		self.lruIndex = prevNext
		self.mappingEntries[prevNext].prevEntry = noEntry // new lru
		entry.nextEntry = noEntry
		entry.prevEntry = self.mruIndex
		if self.mruIndex != noEntry {
			self.mappingEntries[self.mruIndex].nextEntry = index
		}
		self.mruIndex = index
	} else {
		if self.mruIndex != noEntry {
			self.mappingEntries[self.mruIndex].nextEntry = index
		}
		self.mappingEntries[prevPrev].nextEntry = prevNext
		self.mappingEntries[prevNext].prevEntry = prevPrev
		entry.nextEntry = noEntry
		entry.prevEntry = self.mruIndex
		self.mruIndex = index
	}
}
