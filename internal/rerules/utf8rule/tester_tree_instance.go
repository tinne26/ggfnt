package utf8rule

type TesterTreeInstance struct {
	DecisionTree

	Condition uint8
	ConditionIsCached bool
	ConditionCachedValue bool
	NeedsResync bool
}

func (self *TesterTreeInstance) Reconfigure(condition uint8) {
	self.Condition = condition
	self.ConditionIsCached = false
	self.ConditionCachedValue = false
	self.NeedsResync = true
}
