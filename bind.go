package container

import "reflect"

// BindValue bind a value to container
func (impl *containerImpl) BindValue(key string, value interface{}) error {
	return impl.bindValueOverride(key, value, false)
}

// HasBoundValue return whether the kay has bound to a value
func (impl *containerImpl) HasBoundValue(key string) bool {
	impl.lock.RLock()
	defer impl.lock.RUnlock()

	_, ok := impl.objects[key]
	return ok
}

func (impl *containerImpl) bindValueOverride(key string, value interface{}, override bool) error {
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
		override:       override,
		index:          len(impl.objectSlices),
		c:              impl,
		prototype:      false,
	}

	if original, ok := impl.objects[key]; ok {
		if !original.override {
			return buildRepeatedBindError("key repeated, override is not allowed for this key")
		}

		entity.index = original.index
		impl.objects[key] = &entity
		impl.objectSlices[original.index] = &entity

		return nil
	}

	impl.objects[key] = &entity
	impl.objectSlices = append(impl.objectSlices, &entity)

	return nil
}

// BindValueOverride bind a value to container, if key already exist, then replace it
func (impl *containerImpl) BindValueOverride(key string, value interface{}) error {
	return impl.bindValueOverride(key, value, true)
}

// MustBindValueOverride bind a value to container, if key already exist, then replace it, if failed, panic it
func (impl *containerImpl) MustBindValueOverride(key string, value interface{}) {
	impl.Must(impl.BindValueOverride(key, value))
}

// MustBindValue bind a value to container, if failed, panic it
func (impl *containerImpl) MustBindValue(key string, value interface{}) {
	impl.Must(impl.BindValue(key, value))
}

// HasBound return whether a key's type has bound to an object
func (impl *containerImpl) HasBound(key interface{}) bool {
	keyTyp := reflect.ValueOf(key).Type()

	impl.lock.RLock()
	defer impl.lock.RUnlock()

	_, ok := impl.objects[keyTyp]
	return ok
}

// BindWithKey bind a initialize for object with a key
// initialize func(...) (value, error)
func (impl *containerImpl) BindWithKey(key interface{}, initialize interface{}, prototype bool, override bool) error {
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
func (impl *containerImpl) MustBindWithKey(key interface{}, initialize interface{}, prototype bool, override bool) {
	impl.Must(impl.BindWithKey(key, initialize, prototype, override))
}

// Bind bind a initialize for object
// initialize func(...) (value, error)
func (impl *containerImpl) Bind(initialize interface{}, prototype bool, override bool) error {
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
func (impl *containerImpl) MustBind(initialize interface{}, prototype bool, override bool) {
	impl.Must(impl.Bind(initialize, prototype, override))
}

func (impl *containerImpl) bindWithOverride(key interface{}, typ reflect.Type, initialize interface{}, prototype bool, override bool) error {
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

	if original, ok := impl.objects[key]; ok {
		if !original.override {
			return buildRepeatedBindError("key repeated, override is not allowed for this key")
		}

		entity.index = original.index
		impl.objects[key] = entity
		impl.objectSlices[original.index] = entity

		return nil
	}

	entity.index = len(impl.objectSlices)

	impl.objects[key] = entity
	impl.objectSlices = append(impl.objectSlices, entity)

	return nil
}
