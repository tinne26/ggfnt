package rerules

import "errors"

const TesterTooManyRules = "Tester can't have more than 254 rules" // easy limit to bypass, though

var ErrInvalidReRule = errors.New("invalid rewrite rule")
var ErrTesterTooManyRules = errors.New(TesterTooManyRules)
