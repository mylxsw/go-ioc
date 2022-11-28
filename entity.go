package ioc

import (
	"fmt"
	"reflect"
	"sync"
)

// Entity represent an entity in container
type Entity struct {
	lock sync.RWMutex

	key            any          // entity key
	initializeFunc any          // initializeFunc is a func to initialize entity
	value          any          // the value of initializeFunc
	typ            reflect.Type // the type of value
	overridable    bool         // identify whether the entity can be overridden

	prototype bool
	c         *container
}

// Value instance value if not initialized
func (e *Entity) Value(provider EntitiesProvider) (interface{}, error) {
	if e.prototype {
		return e.createValue(provider)
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	if e.value == nil {
		val, err := e.createValue(provider)
		if err != nil {
			return nil, err
		}

		e.value = val
	}

	return e.value, nil
}

func (e *Entity) createValue(provider EntitiesProvider) (interface{}, error) {
	initializeValue := reflect.ValueOf(e.initializeFunc)
	argValues, err := e.c.funcArgs(initializeValue.Type(), provider)
	if err != nil {
		return nil, err
	}

	returnValues := reflect.ValueOf(e.initializeFunc).Call(argValues)
	if len(returnValues) <= 0 {
		return nil, buildInvalidReturnValueCountError("expect greater than 0, got 0")
	}

	if len(returnValues) > 1 && !returnValues[1].IsNil() && returnValues[1].Interface() != nil {
		if err, ok := returnValues[1].Interface().(error); ok {
			return nil, fmt.Errorf("(%s) %w", e.key, err)
		}

		// 如果第二个返回值不是 error，则强制转换为 error
		return nil, fmt.Errorf("(%s) %v", e.key, returnValues[1].Interface())
	}

	return returnValues[0].Interface(), nil
}
