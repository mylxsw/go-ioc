package container

import "reflect"

type Conditional interface {
	GetInitFunc() interface{}
	Matched(cc Container) (bool, error)
}

type conditional struct {
	init interface{}
	on   interface{}
}

func WithCondition(init interface{}, onCondition interface{}) Conditional {
	onType := reflect.TypeOf(onCondition)
	argCount := onType.NumOut()
	if argCount != 1 && argCount != 2 {
		panic("invalid argument onCondition: onCondition() bool or onCondition() (bool, error)")
	}

	return conditional{init: init, on: onCondition}
}

func (cond conditional) GetInitFunc() interface{} {
	return cond.init
}

func (cond conditional) Matched(cc Container) (bool, error) {
	res, err := cc.Call(cond.on)
	if err != nil {
		return false, err
	}

	if len(res) == 2 {
		matched, err := res[0], res[1]
		return matched.(bool), err.(error)
	}

	return res[0].(bool), nil
}
