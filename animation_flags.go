package ggfnt

type AnimationFlags uint8
const (
	AnimFlagLoopable AnimationFlags = 0b0000_0001 // can wrap back to start after reaching the end
	AnimFlagSequential AnimationFlags = 0b0000_0010 // should play from start to finish in order, or the reverse
	AnimFlagTerminal AnimationFlags = 0b0000_0100 // animation shouldn't be rewinded/replayed automatically
	AnimFlagSplit AnimationFlags = 0b0000_1000 // frames are independent and we can stop/rest at any of them
	AnimFlagsGroupMask AnimationFlags = 0b1110_0000 // usage still undefined. maybe better leave as custom use flags?
)
