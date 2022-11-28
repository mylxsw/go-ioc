package ioc

import "reflect"

// BindValue bind a value to container
func (impl *container) BindValue(key string, value interface{}) error {
	return impl.bindValueOverride(key, value, false)
}

// HasBoundValue return whether the kay has bound to a value
func (impl *container) HasBoundValue(key string) bool {
	impl.lock.RLock()
	defer impl.lock.RUnlock()

	_, ok := impl.entities[key]
	return ok
}

func (impl *container) bindValueOverride(key string, value interface{}, override bool) error {
	if value == nil {
		return buildInvalidArgsError("value is nil")
	}

	if key == "" || key == "@" {
		return buildInvalidArgsError("key can not be empty or reserved words(@)")
	}

	impl.lock.Lock()
	defer impl.lock.Unlock()

	entity := Entity{
		initializeFunc: nil,
		key:            key,
		typ:            reflect.TypeOf(value),
		value:          value,
		overridable:    override,
		c:              impl,
		prototype:      false,
	}

	if v, ok := impl.entities[key]; ok {
		if !v.overridable {
			return buildRepeatedBindError("key repeated, overridable is not allowed for this key")
		}

		impl.entities[key] = &entity
		return nil
	}

	impl.entities[key] = &entity

	return nil
}

// BindValueOverride bind a value to container, if key already exist, then replace it
func (impl *container) BindValueOverride(key string, value interface{}) error {
	return impl.bindValueOverride(key, value, true)
}

// MustBindValueOverride bind a value to container, if key already exist, then replace it, if failed, panic it
func (impl *container) MustBindValueOverride(key string, value interface{}) {
	impl.Must(impl.BindValueOverride(key, value))
}

// MustBindValue bind a value to container, if failed, panic it
func (impl *container) MustBindValue(key string, value interface{}) {
	impl.Must(impl.BindValue(key, value))
}

// HasBound return whether a key's type has bound to an object
func (impl *container) HasBound(key interface{}) bool {
	keyTyp := reflect.ValueOf(key).Type()

	impl.lock.RLock()
	defer impl.lock.RUnlock()

	_, ok := impl.entities[keyTyp]
	return ok
}

// BindWithKey bind a initialize for object with a key
// initialize func(...) (value, error)
func (impl *container) BindWithKey(key interface{}, initialize interface{}, prototype bool, override bool) error {
	if _, ok := initialize.(Conditional); !ok {
		initialize = WithCondition(initialize, func() bool { return true })
	}

	initF := initialize.(Conditional).getInitFunc()

	if !reflect.ValueOf(initF).IsValid() {
		return buildInvalidArgsError("initialize is nil")
	}

	if err := impl.isValidKeyKind(reflect.TypeOf(key).Kind()); err != nil {
		return err
	}

	initializeType := reflect.ValueOf(initF).Type()
	if initializeType.Kind() == reflect.Func {
		if initializeType.NumOut() <= 0 {
			return buildInvalidArgsError("expect func return values count greater than 0, but got 0")
		}

		return impl.bindWithOverride(key, initializeType.Out(0), initialize, prototype, override)
	}

	initFunc := WithCondition(func() interface{} { return initF }, initialize.(Conditional).matched)
	return impl.bindWithOverride(key, initializeType, initFunc, prototype, override)
}

// MustBindWithKey bind a initialize for object with a key, if failed then panic
func (impl *container) MustBindWithKey(key interface{}, initialize interface{}, prototype bool, override bool) {
	impl.Must(impl.BindWithKey(key, initialize, prototype, override))
}

// Bind bind a initialize for object
// initialize func(...) (value, error)
func (impl *container) Bind(initialize interface{}, prototype bool, override bool) error {
	if _, ok := initialize.(Conditional); !ok {
		initialize = conditional{init: initialize, on: func() bool { return true }}
	}

	initF := initialize.(Conditional).getInitFunc()

	if !reflect.ValueOf(initF).IsValid() {
		return buildInvalidArgsError("initialize is nil")
	}

	initializeType := reflect.ValueOf(initF).Type()
	if initializeType.Kind() == reflect.Func {
		if initializeType.NumOut() <= 0 {
			return buildInvalidArgsError("expect func return values count greater than 0, but got 0")
		}

		typ := initializeType.Out(0)

		if err := impl.isValidKeyKind(typ.Kind()); err != nil {
			return err
		}

		return impl.bindWithOverride(typ, typ, initialize, prototype, override)
	}

	if err := impl.isValidKeyKind(initializeType.Kind()); err != nil {
		return err
	}

	initFunc := WithCondition(func() interface{} { return initF }, initialize.(Conditional).getOnCondition())
	return impl.bindWithOverride(initializeType, initializeType, initFunc, prototype, override)
}

// MustBind bind a initialize, if failed then panic
func (impl *container) MustBind(initialize interface{}, prototype bool, override bool) {
	impl.Must(impl.Bind(initialize, prototype, override))
}

func (impl *container) bindWithOverride(key interface{}, typ reflect.Type, initialize interface{}, prototype bool, override bool) error {
	var entity *Entity
	if cond, ok := initialize.(Conditional); ok {
		matched, err := cond.matched(impl)
		if err != nil {
			return err
		}

		if !matched {
			return nil
		}

		entity = impl.newEntity(key, typ, cond.getInitFunc(), prototype, override)
	} else {
		entity = impl.newEntity(key, typ, initialize, prototype, override)
	}

	impl.lock.Lock()
	defer impl.lock.Unlock()

	if v, ok := impl.entities[entity.key]; ok {
		if !v.overridable {
			return buildRepeatedBindError("key repeated, overridable is not allowed for this key")
		}

		impl.entities[key] = entity
		return nil
	}

	impl.entities[key] = entity

	return nil
}
