package builder

import "slices"
import "errors"
import "fmt"

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"

type mappingSwitchEntry struct {
	Settings []uint8
}

func (self *Font) AddSwitch(settings ...ggfnt.SettingKey) (uint8, error) {
	if len(settings) < 1 || len(settings) > 255 {
		return 0, errors.New("mapping switch must contain between 1 and 255 settings")
	}
	if len(self.mappingSwitches) >= 254 {
		return 0, errors.New("can't have more than 254 mapping switches")
	}
	repeated := make(map[ggfnt.SettingKey]struct{}, len(settings))
	numSettings := uint8(len(self.settings))
	newSettings := make([]uint8, 0, len(settings))
	for _, setting := range settings {
		if setting >= ggfnt.SettingKey(numSettings) {
			return 0, errors.New("mapping switch contains undefined setting")
		}
		_, alreadyAdded := repeated[setting]
		if alreadyAdded {
			return 0, errors.New("mapping switch can't contain repeated settings")
		}
		repeated[setting] = struct{}{}
		newSettings = append(newSettings, uint8(setting))
	}

	key := uint8(len(self.mappingSwitches))
	self.mappingSwitches = append(self.mappingSwitches, mappingSwitchEntry{ Settings: newSettings })
	return key, nil
}

type mappingEntry struct {
	SwitchType uint8
	SwitchCases []mappingGroup
}

// Before calling this, the caller should cross check switch type with 
// the number of SwitchCases.
func (self *mappingEntry) AppendTo(data []byte, glyphLookup map[uint64]uint16, scratchBuffer []uint16) ([]byte, []uint16, error) {
	if len(scratchBuffer) != 0 { panic(brokenCode) }
	
	var err error
	data = append(data, self.SwitchType)
	
	// single glyph case
	if self.SwitchType == 255 {
		if len(self.SwitchCases) != 1 { panic(invalidInternalState) }
		if len(self.SwitchCases[0].Glyphs) != 1 { panic(invalidInternalState) }
		glyphIndex, found := glyphLookup[self.SwitchCases[0].Glyphs[0]]
		if !found { panic(invalidInternalState) }
		return internal.AppendUint16LE(data, glyphIndex), scratchBuffer, nil
	} else if self.SwitchType == 254 {
		if len(self.SwitchCases) != 1 { panic(invalidInternalState) }
		if len(self.SwitchCases[0].Glyphs) <= 1 { panic(invalidInternalState) }
		return self.SwitchCases[0].AppendTo(data, glyphLookup, scratchBuffer)
	}

	// more involved switch case
	for i, _ := range self.SwitchCases {
		data, scratchBuffer, err = self.SwitchCases[i].AppendTo(data, glyphLookup, scratchBuffer)
		if err != nil { return data, scratchBuffer[ : 0], err }
		if len(data) > ggfnt.MaxFontDataSize {
			return data, scratchBuffer[ : 0], errFontDataExceedsMax
		}
	}
	return data, scratchBuffer[ : 0], nil
}

type mappingGroup struct {
	Glyphs []uint64
	AnimationFlags ggfnt.AnimationFlags
}

func (self *mappingGroup) AppendTo(data []byte, glyphLookup map[uint64]uint16, scratchBuffer []uint16) ([]byte, []uint16, error) {
	if len(self.Glyphs) == 0 || len(self.Glyphs) > 128 { panic(invalidInternalState) }
	
	// single glyph case
	if len(self.Glyphs) == 1 {
		glyphIndex, found := glyphLookup[self.Glyphs[0]]
		if !found { panic(invalidInternalState) }
		data = append(data, 0) // list of 1 glyph
		return internal.AppendUint16LE(data, glyphIndex), scratchBuffer, nil
	}

	// get actual glyph indices
	scratchBuffer = internal.SetSliceSize(scratchBuffer, len(self.Glyphs))
	for i := 0; i < len(self.Glyphs); i++ {
		glyphIndex, found := glyphLookup[self.Glyphs[i]]
		if !found { panic(invalidInternalState) }
		scratchBuffer[i] = glyphIndex
	}

	// sort scratch buffer, makes it easier to see if glyphs are consecutive
	slices.Sort(scratchBuffer)
	if isContinuousSlice(scratchBuffer) {
		data = append(data, 0b1000_0000 | uint8(len(self.Glyphs) - 1))
		data = append(data, uint8(self.AnimationFlags))
		data = internal.AppendUint16LE(data, uint16(scratchBuffer[0]))
	} else {
		data = append(data, uint8(len(self.Glyphs) - 1))
		data = append(data, uint8(self.AnimationFlags))
		for _, glyphIndex := range scratchBuffer {
			data = internal.AppendUint16LE(data, uint16(glyphIndex))
		}
	}

	return data, scratchBuffer[ : 0], nil
}


// --- main mapping API ---

func (self *Font) Map(codePoint rune, glyphUID uint64) error {
	// validation
	if codePoint < ' ' {
		return errors.New("can't map code points before ' ' (space)") 
	}
	_, hasData := self.glyphData[glyphUID]
	if !hasData {
		return errors.New("attempted to map '" + string(codePoint) + "' to an undefined glyph")
	}

	// actual addition
	self.runeMapping[codePoint] = mappingEntry{
		SwitchType: 255,
		SwitchCases: []mappingGroup{ mappingGroup{ Glyphs: []uint64{glyphUID} } },
	}
	return nil
}

func (self *Font) MapGroup(codePoint rune, animFlags ggfnt.AnimationFlags, glyphUIDs ...uint64) error {
	// basic validation
	if codePoint < ' ' {
		return errors.New("can't map code points before ' ' (space)") 
	}
	if len(glyphUIDs) < 2 {
		return errors.New("mapping a glyph group to a code point requires at least 2 glyphs")
	}
	if len(glyphUIDs) > 127 {
		return errors.New("glyph groups can't exceed 127 glyphs")
	}

	// glyph UIDs validation
	err := self.validateMapGlyphs(codePoint, glyphUIDs...)
	if err != nil { return err }

	// actual addition
	self.runeMapping[codePoint] = mappingEntry{
		SwitchType: 254,
		SwitchCases: []mappingGroup{ mappingGroup{ Glyphs: glyphUIDs, AnimationFlags: animFlags } },
	}
	return nil
}

func (self *Font) validateMapGlyphs(codePoint rune, glyphUIDs ...uint64) error {
	for _, glyphUID := range glyphUIDs {
		_, hasData := self.glyphData[glyphUID]
		if !hasData {
			return errors.New("attempted to map '" + string(codePoint) + "' to an undefined glyph")
		}
	}
	return nil
}

func (self *Font) MapWithSwitchSingles(codePoint rune, mapSwitch uint8, glyphUIDs ...uint64) error {
	// basic validation
	if codePoint < ' ' {
		return errors.New("can't map code points before ' ' (space)")
	}
	if mapSwitch >= 254 {
		panic("MapWithSwitch expects a map switch < 254")
	}
	if int(mapSwitch) >= len(self.mappingSwitches) {
		return errors.New("can't map with undefined switch")
	}
	numSwitchCases := self.computeNumSwitchCases(mapSwitch)
	if len(glyphUIDs) != numSwitchCases {
		return fmt.Errorf("switch %d expects %d glyph groups, but received %d", mapSwitch, numSwitchCases, len(glyphUIDs))
	}
	
	cases := make([]mappingGroup, 0, len(glyphUIDs))
	for _, glyphUID := range glyphUIDs {
		cases = append(cases, mappingGroup{ Glyphs: []uint64{glyphUID}, AnimationFlags: 0 })
	}
	self.runeMapping[codePoint] = mappingEntry{
		SwitchType: mapSwitch,
		SwitchCases: cases,
	}
	return nil
}

// TODO: I also need removal, edit (modify) and get. messy.
func (self *Font) MapWithSwitch(codePoint rune, mapSwitch uint8, glyphUIDs [][]uint64, animFlags []ggfnt.AnimationFlags) error {
	// basic validation
	if codePoint < ' ' {
		return errors.New("can't map code points before ' ' (space)")
	}
	if mapSwitch >= 254 {
		panic("MapWithSwitch expects a map switch < 254")
	}
	if int(mapSwitch) >= len(self.mappingSwitches) {
		return errors.New("can't map with undefined switch")
	}
	numSwitchCases := self.computeNumSwitchCases(mapSwitch)
	if len(glyphUIDs) != numSwitchCases {
		return fmt.Errorf("switch %d expects %d glyph groups, but received %d", mapSwitch, numSwitchCases, len(glyphUIDs))
	}
	finalAnimFlags := make([]ggfnt.AnimationFlags, 0, len(glyphUIDs))
	animFlagIndex := 0
	for _, group := range glyphUIDs {
		if len(group) == 0 { panic("glyph groups can't be empty") }
		if len(group) == 1 { continue }
		if len(animFlags) <= animFlagIndex { return errors.New("not enough animation flags for all multi-glyph groups") }
		finalAnimFlags = append(finalAnimFlags, animFlags[animFlagIndex])
		animFlagIndex += 1
	}
	if animFlagIndex != len(animFlags) {
		return errors.New("number of animation flags doesn't match number of multi-glyph groups")
	}
	
	cases := make([]mappingGroup, 0, len(glyphUIDs))
	for i, group := range glyphUIDs {
		cases = append(cases, mappingGroup{ Glyphs: group, AnimationFlags: finalAnimFlags[i] })
	}
	self.runeMapping[codePoint] = mappingEntry{
		SwitchType: mapSwitch,
		SwitchCases: cases,
	}
	return nil
}

func (self *Font) Unmap(codePoint rune) error {
	panic("unimplemented")
}

