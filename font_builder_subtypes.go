package ggfnt

type editionCategory struct {
	Name string
	Size uint16
}

type editionKerningPair struct {
	Pair uint32
	Class uint16
}

type editionKerningClass struct {
	Name string
	Value int8
}

type nameIndexEntry struct {
	Name string
	Index uint16
}

type variableEntry struct {
	DefaultValue uint8
	MinValue uint8
	MaxValue uint8
}
