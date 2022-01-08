package container

import (
	"errors"
	"reflect"
)

type Conditional interface {
	getInitFunc() interface{}
	getOnCondition() interface{}
	matched(cc Container) (bool, error)
}

type conditional struct {
	init interface{}
	on   interface{}
}

// WithCondition 创建 Conditional 接口实例
// init 参数为传递个 Singleton/Prototype 方法的实例创建方法
// onCondition 参数支持两种形式
// 	- `onCondition(依赖注入参数列表...) bool`
//	- `onCondition(依赖注入参数列表...) (bool, error)`
func WithCondition(init interface{}, onCondition interface{}) Conditional {
	if onCondition == nil {
		panic("invalid argument onCondition: can not be nil [onCondition() bool or onCondition() (bool, error)]")
	}
	onType := reflect.TypeOf(onCondition)
	if onType.Kind() != reflect.Func {
		panic("invalid argument onCondition: must be a func [onCondition() bool or onCondition() (bool, error)]")
	}

	argCount := onType.NumOut()
	if argCount != 1 && argCount != 2 {
		panic("invalid argument onCondition: onCondition() bool or onCondition() (bool, error)")
	}

	if onType.Out(0).Kind() != reflect.Bool {
		panic("invalid argument onCondition: return value must be bool [onCondition() bool or onCondition() (bool, error)]")
	}

	// onCondition() (bool, error)
	if argCount == 2 {
		if onType.Out(1) != reflect.TypeOf(errors.New("")) {
			panic("invalid argument onCondition: the second return value must be error [onCondition() (bool, error)]")
		}
	}

	return conditional{init: init, on: onCondition}
}

func (cond conditional) getInitFunc() interface{} {
	return cond.init
}

func (cond conditional) getOnCondition() interface{} {
	return cond.on
}

func (cond conditional) matched(cc Container) (bool, error) {
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
