package mask

func doesPanic(function func()) (didPanic bool) {
	didPanic = false
	defer func() { didPanic = (recover() != nil) }()
	function()
	return
}

func doesNotPanic(function func()) (didNotPanic bool) {
	didNotPanic = true
	defer func() { didNotPanic = (recover() == nil) }()
	function()
	return
}
