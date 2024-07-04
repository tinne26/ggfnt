package ggfnt

// TODO: unimplemented, and needs new strategy, tests, etc.
// NOTE: actually, it's unclear what I'm doing here. I can
//       cache offsets to cached conditions, but those aren't
//       even super expensive to iterate right away in most
//       cases, so I have to be careful, because a naive approach
//       might actually end up making things more expensive
//       instead.
// Do we need every code point mapped? Or only the ones that
// have <254 conditions and cases?

const noEntry uint32 = 0xFFFF_FFFF

type mappingEntry struct {
	dropSignature uint32
	mappingOffset uint32
	prevEntry uint32
	nextEntry uint32
}

// Glyph mapping cache. These are recommended to be used at small sizes,
// like 192. In general, a glyph mapping cache is not even critical for
// operation unless the font has multiple settings affecting mapping,
// animations and so on. In those cases, the mapping can get more expensive,
// and using a mapping cache can be quite beneficial.
type MappingCache struct {
	font *Font
	dropCounter uint32
	mruIndex uint32
	lruIndex uint32
	cachedMappings map[rune]uint32
	mappingEntries []mappingEntry
}

// The size is statically allocated.
func NewMappingCache(font *Font, size int) *MappingCache {
	// safety assertions
	if size <= 0 { panic("mapping cache size must be positive") }
	if size >= int(noEntry) { panic("mapping cache size is way too big") }
	if font == nil { panic("mapping cache can't accept nil font") }

	return &MappingCache{
		font: font, 
		mruIndex: noEntry,
		lruIndex: noEntry,
		cachedMappings: make(map[rune]uint32, size),
		mappingEntries: make([]mappingEntry, size),
	}
}

// Drops must be manually requested due to variable changes.
func (self *MappingCache) Drop() {
	self.dropCounter += 1
}

// Returned glyph index will be [GlyphMissing] if not found.
func (self *MappingCache) Get(codePoint rune, picker func(uint8) uint8) GlyphIndex {
	entryIndex, found := self.cachedMappings[codePoint]
	if found { // easy case
		entry := &self.mappingEntries[entryIndex]
		// TODO: update mru
		if entry.dropSignature != self.dropCounter {
			// TODO: get num choices
			// self.font. entry.mappingOffset
			var compressedRange bool
			var numChoices uint8
			choice := picker(numChoices)
			_ = choice
			if compressedRange {
				// return font choices [0] + GlyphIndex(choice)
			} else {
				// return font choices [choice]
			}
		} else {
			// signature is invalid, find new data and replace this very entry
			// self.mappingEntries[entryIndex] = newEntry
		}
	} else { // entry index not found
		// evict lru
		// ...

		// create new entry
		// ...

		// set as mru
		// ...
	}

	panic("unimplemented")
}
