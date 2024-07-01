package rerules

import "errors"

import "github.com/tinne26/ggfnt"

const TesterTooManyRules = "Tester can't have more than 254 rules" // easy limit to bypass, though

var ErrInvalidReRule = errors.New("invalid rewrite rule")
var ErrTesterTooManyRules = errors.New(TesterTooManyRules)

// errWithRule, hasRule := err.(interface { Rule() ggfnt.Utf8RewriteRule })
// if hasRule { rule := errWithRule.Rule() ; panic(rule.String()) }
type invalidUtf8Rule struct {
	rule ggfnt.Utf8RewriteRule
}
func (self invalidUtf8Rule) Error() string {
	return "invalid rewrite rule"
}
func (self invalidUtf8Rule) Rule() ggfnt.Utf8RewriteRule {
	return self.rule
}
func InvalidUtf8RuleErr(rule ggfnt.Utf8RewriteRule) error {
	return invalidUtf8Rule{ rule }
}
