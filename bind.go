package container

import "reflect"

// BindValue bind a value to container
func (c *containerImpl) BindValue(key string, value interface{}) error {
	return c.bindValueOverride(key, value, false)
}

// HasBoundValue return whether the kay has bound to a value
func (c *containerImpl) HasBoundValue(key string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.objects[key]
	return ok
}

func (c *containerImpl) bindValueOverride(key string, value interface{}, override bool) error {
	if value == nil {
		return buildInvalidArgsError("value is nil")
	}

	if key == "" || key == "@" {
		return buildInvalidArgsError("key can not be empty or reserved words(@)")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	entity := Entity{
		initializeFunc: nil,
		key:            key,
		typ:            reflect.TypeOf(value),
		value:          value,
		override:       override,
		index:          len(c.objectSlices),
		c:              c,
		prototype:      false,
	}

	if original, ok := c.objects[key]; ok {
		if !original.override {
			return buildRepeatedBindError("key repeated, override is not allowed for this key")
		}

		entity.index = original.index
		c.objects[key] = &entity
		c.objectSlices[original.index] = &entity

		return nil
	}

	c.objects[key] = &entity
	c.objectSlices = append(c.objectSlices, &entity)

	return nil
}

// BindValueOverride bind a value to container, if key already exist, then replace it
func (c *containerImpl) BindValueOverride(key string, value interface{}) error {
	return c.bindValueOverride(key, value, true)
}

// MustBindValueOverride bind a value to container, if key already exist, then replace it, if failed, panic it
func (c *containerImpl) MustBindValueOverride(key string, value interface{}) {
	c.Must(c.BindValueOverride(key, value))
}

// MustBindValue bind a value to container, if failed, panic it
func (c *containerImpl) MustBindValue(key string, value interface{}) {
	c.Must(c.BindValue(key, value))
}

// HasBound return whether a key's type has bound to an object
func (c *containerImpl) HasBound(key interface{}) bool {
	keyTyp := reflect.ValueOf(key).Type()

	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.objects[keyTyp]
	return ok
}

// BindWithKey bind a initialize for object with a key
// initialize func(...) (value, error)
func (c *containerImpl) BindWithKey(key interface{}, initialize interface{}, prototype bool, override bool) error {
	if !reflect.ValueOf(initialize).IsValid() {
		return buildInvalidArgsError("initialize is nil")
	}

	if err := c.isValidKeyKind(reflect.TypeOf(key).Kind()); err != nil {
		return err
	}

	initializeType := reflect.ValueOf(initialize).Type()
	if initializeType.Kind() == reflect.Func {
		if initializeType.NumOut() <= 0 {
			return buildInvalidArgsError("expect func return values count greater than 0, but got 0")
		}

		return c.bindWithOverride(key, initializeType.Out(0), initialize, prototype, override)
	}

	initFunc := func() interface{} { return initialize }
	return c.bindWithOverride(key, initializeType, initFunc, prototype, override)
}

// MustBindWithKey bind a initialize for object with a key, if failed then panic
func (c *containerImpl) MustBindWithKey(key interface{}, initialize interface{}, prototype bool, override bool) {
	c.Must(c.BindWithKey(key, initialize, prototype, override))
}

// Bind bind a initialize for object
// initialize func(...) (value, error)
func (c *containerImpl) Bind(initialize interface{}, prototype bool, override bool) error {
	if !reflect.ValueOf(initialize).IsValid() {
		return buildInvalidArgsError("initialize is nil")
	}

	initializeType := reflect.ValueOf(initialize).Type()
	if initializeType.Kind() == reflect.Func {
		if initializeType.NumOut() <= 0 {
			return buildInvalidArgsError("expect func return values count greater than 0, but got 0")
		}

		typ := initializeType.Out(0)

		if err := c.isValidKeyKind(typ.Kind()); err != nil {
			return err
		}

		return c.bindWithOverride(typ, typ, initialize, prototype, override)
	}

	if err := c.isValidKeyKind(initializeType.Kind()); err != nil {
		return err
	}

	initFunc := func() interface{} { return initialize }
	return c.bindWithOverride(initializeType, initializeType, initFunc, prototype, override)
}

// MustBind bind a initialize, if failed then panic
func (c *containerImpl) MustBind(initialize interface{}, prototype bool, override bool) {
	c.Must(c.Bind(initialize, prototype, override))
}


func (c *containerImpl) bindWithOverride(key interface{}, typ reflect.Type, initialize interface{}, prototype bool, override bool) error {
	entity := c.newEntity(key, typ, initialize, prototype, override)

	c.lock.Lock()
	defer c.lock.Unlock()

	if original, ok := c.objects[key]; ok {
		if !original.override {
			return buildRepeatedBindError("key repeated, override is not allowed for this key")
		}

		entity.index = original.index
		c.objects[key] = entity
		c.objectSlices[original.index] = entity

		return nil
	}

	entity.index = len(c.objectSlices)

	c.objects[key] = entity
	c.objectSlices = append(c.objectSlices, entity)

	return nil
}

func (c *containerImpl) bindWith(key interface{}, typ reflect.Type, initialize interface{}, prototype bool) error {
	return c.bindWithOverride(key, typ, initialize, prototype, false)
}