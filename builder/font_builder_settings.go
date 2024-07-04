package builder

import "errors"

import "github.com/tinne26/ggfnt"
import "github.com/tinne26/ggfnt/internal"

type settingEntry struct {
	Name string
	Options []string
}

func (self *settingEntry) AppendWords(words map[string]int16) {
	for _, opt := range self.Options {
		_, hasWord := words[opt]
		if !hasWord {
			words[opt] = 0
		}
	}
}

func organizeWords(builtinWords, words map[string]int16) error {
	if len(words) > 255 { return errors.New("setting options require too many words") }

	// set indices for all unique words
	var customWordIndex uint16
	var minBuiltinIndex uint16 = 65535
	for word, _ := range words {
		builtinIndex, isBuiltIn := builtinWords[word]
		if isBuiltIn {
			words[word] = -builtinIndex
			if uint16(builtinIndex) < minBuiltinIndex {
				minBuiltinIndex = uint16(builtinIndex)
			}
		} else {
			words[word] = int16(customWordIndex)
			customWordIndex += 1
		}
	}

	// see if builtin words can be used or not
	for minBuiltinIndex < customWordIndex {
		target := -int16(minBuiltinIndex)
		minBuiltinIndex = 65535
		for word, index := range words {
			if index == target {
				words[word] = int16(customWordIndex)
				customWordIndex += 1
			} else if index < 0 { // built-in word
				if uint16(-index) < minBuiltinIndex {
					minBuiltinIndex = uint16(-index)
				}
			}
		}
	}

	// delete built-in words
	for word, index := range words {
		if index < 0 { delete(words, word) }
	}
	
	// NOTE: sorting is done later to improve encoding
	// stability, but directly on the encoding code

	return nil
}

func (self *settingEntry) AppendTo(data []byte, builtinWords, words map[string]int16) []byte {
	for _, opt := range self.Options {
		wordIndex, found := words[opt]
		if !found {
			wordIndex, found = builtinWords[opt]
			if !found { panic(invalidInternalState) }
		}
		if wordIndex > 255 { panic(invalidInternalState) }
		data = append(data, uint8(wordIndex))
	}
	return data
}

func (self *Font) AddSetting(name string, options ...string) (ggfnt.SettingKey, error) {
	err := internal.ValidateBasicName(name)
	if err != nil { return 0, err }
	for _, opt := range options {
		err := internal.ValidateBasicName(opt)
		if err != nil { return 0, err }
	}

	if len(options) > 255 {
		return 0, errors.New("setting can't have more than 255 options")
	}

	if len(self.settings) >= 255 {
		return 0, errors.New("can't add more than 255 settings")
	}

	for i, _ := range self.settings {
		if self.settings[i].Name == name {
			return 0, errors.New("can't add two settings with the same name")
		}
	}
	
	key := ggfnt.SettingKey(len(self.settings))
	self.settings = append(self.settings, settingEntry{ Name: name, Options: options })
	return key, nil
}
