package container

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// ResolveError is a error when container can not resolve a object
type ResolveError struct {
	err error
}

func newResolveError(err error) ResolveError {
	return ResolveError{err: err}
}

// Error return the error string
func (re ResolveError) Error() string {
	return re.err.Error()
}

// Entity represent a entity in container
type Entity struct {
	lock sync.RWMutex

	key            interface{} // entity key
	initializeFunc interface{} // initializeFunc is a func to initialize entity
	value          interface{}
	typ            reflect.Type
	index          int // the index in the container

	prototype bool
	c         *Container
}

// Value instance value if not initialized
func (e *Entity) Value(provider func() []*Entity) (interface{}, error) {
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

func (e *Entity) createValue(provider func() []*Entity) (interface{}, error) {
	initializeValue := reflect.ValueOf(e.initializeFunc)
	argValues, err := e.c.funcArgs(initializeValue.Type(), provider)
	if err != nil {
		return nil, err
	}

	returnValues := reflect.ValueOf(e.initializeFunc).Call(argValues)
	if len(returnValues) <= 0 {
		return nil, ErrInvalidReturnValueCount("expect greater than 0, got 0")
	}

	if len(returnValues) > 1 && !returnValues[1].IsNil() && returnValues[1].Interface() != nil {
		err, ok := returnValues[1].Interface().(error)
		if ok {
			return nil, err
		}
	}

	return returnValues[0].Interface(), nil
}

// Container is a dependency injection container
type Container struct {
	lock sync.RWMutex

	objects      map[interface{}]*Entity
	objectSlices []*Entity
}

// New create a new container
func New() *Container {
	cc := &Container{
		objects:      make(map[interface{}]*Entity),
		objectSlices: make([]*Entity, 0),
	}

	cc.MustSingleton(func() *Container {
		return cc
	})

	cc.MustSingleton(func() context.Context {
		return context.Background()
	})

	return cc
}

// NewWithContext create a new container with context support
func NewWithContext(ctx context.Context) *Container {
	cc := &Container{
		objects:      make(map[interface{}]*Entity),
		objectSlices: make([]*Entity, 0),
	}

	cc.MustSingleton(func() *Container {
		return cc
	})

	cc.MustSingleton(func() context.Context {
		return ctx
	})

	return cc
}

// Must if err is not nil, panic it
func (c *Container) Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Prototype bind a prototype
// initialize func(...) (value, error)
func (c *Container) Prototype(initialize interface{}) error {
	return c.Bind(initialize, true)
}

// MustPrototype bind a prototype, if failed then panic
func (c *Container) MustPrototype(initialize interface{}) {
	c.Must(c.Prototype(initialize))
}

// PrototypeWithKey bind a prototype with key
// initialize func(...) (value, error)
func (c *Container) PrototypeWithKey(key interface{}, initialize interface{}) error {
	return c.BindWithKey(key, initialize, true)
}

// MustPrototypeWithKey bind a prototype with key, it failed, then panic
func (c *Container) MustPrototypeWithKey(key interface{}, initialize interface{}) {
	c.Must(c.PrototypeWithKey(key, initialize))
}

// Singleton bind a singleton
// initialize func(...) (value, error)
func (c *Container) Singleton(initialize interface{}) error {
	return c.Bind(initialize, false)
}

// MustSingleton bind a singleton, if bind failed, then panic
func (c *Container) MustSingleton(initialize interface{}) {
	c.Must(c.Singleton(initialize))
}

// SingletonWithKey bind a singleton with key
// initialize func(...) (value, error)
func (c *Container) SingletonWithKey(key interface{}, initialize interface{}) error {
	return c.BindWithKey(key, initialize, false)
}

// MustSingletonWithKey bind a singleton with key, if failed, then panic
func (c *Container) MustSingletonWithKey(key interface{}, initialize interface{}) {
	c.Must(c.SingletonWithKey(key, initialize))
}

// BindValue bind a value to container
func (c *Container) BindValue(key interface{}, value interface{}) error {
	if value == nil {
		return ErrInvalidArgs("value is nil")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.objects[key]; ok {
		return ErrRepeatedBind("key repeated")
	}

	entity := Entity{
		initializeFunc: nil,
		key:            key,
		typ:            reflect.TypeOf(value),
		value:          value,
		index:          len(c.objectSlices),
		c:              c,
		prototype:      false,
	}

	c.objects[key] = &entity
	c.objectSlices = append(c.objectSlices, &entity)

	return nil
}

// MustBindValue bind a value to container, if failed, panic it
func (c *Container) MustBindValue(key interface{}, value interface{}) {
	c.Must(c.BindValue(key, value))
}

// ServiceProvider create a provider from initializes
func (c *Container) ServiceProvider(initializes ...interface{}) (func() []*Entity, error) {
	entities := make([]*Entity, len(initializes))
	for i, init := range initializes {
		entity, err := c.NewEntity(init, false)
		if err != nil {
			return nil, err
		}

		entities[i] = entity
	}

	return func() []*Entity {
		return entities
	}, nil
}

// NewEntity create a new entity
func (c *Container) NewEntity(initialize interface{}, prototype bool) (*Entity, error) {
	if !reflect.ValueOf(initialize).IsValid() {
		return nil, ErrInvalidArgs("initialize is nil")
	}

	initializeType := reflect.ValueOf(initialize).Type()
	if initializeType.NumOut() <= 0 {
		return nil, ErrInvalidArgs("expect func return values count greater than 0, but got 0")
	}

	typ := initializeType.Out(0)
	return c.newEntity(typ, typ, initialize, prototype), nil
}

func (c *Container) newEntity(key interface{}, typ reflect.Type, initialize interface{}, prototype bool) *Entity {
	entity := Entity{
		initializeFunc: initialize,
		key:            key,
		typ:            typ,
		value:          nil,
		c:              c,
		prototype:      prototype,
	}

	return &entity
}

// Bind bind a initialize for object
// initialize func(...) (value, error)
func (c *Container) Bind(initialize interface{}, prototype bool) error {
	if !reflect.ValueOf(initialize).IsValid() {
		return ErrInvalidArgs("initialize is nil")
	}

	initializeType := reflect.ValueOf(initialize).Type()
	if initializeType.NumOut() <= 0 {
		return ErrInvalidArgs("expect func return values count greater than 0, but got 0")
	}

	typ := initializeType.Out(0)
	return c.bindWith(typ, typ, initialize, prototype)
}

// MustBind bind a initialize, if failed then panic
func (c *Container) MustBind(initialize interface{}, prototype bool) {
	c.Must(c.Bind(initialize, prototype))
}

// BindWithKey bind a initialize for object with a key
// initialize func(...) (value, error)
func (c *Container) BindWithKey(key interface{}, initialize interface{}, prototype bool) error {
	if !reflect.ValueOf(initialize).IsValid() {
		return ErrInvalidArgs("initialize is nil")
	}

	initializeType := reflect.ValueOf(initialize).Type()
	if initializeType.NumOut() <= 0 {
		return ErrInvalidArgs("expect func return values count greater than 0, but got 0")
	}

	return c.bindWith(key, initializeType.Out(0), initialize, prototype)
}

// MustBindWithKey bind a initialize for object with a key, if failed then panic
func (c *Container) MustBindWithKey(key interface{}, initialize interface{}, prototype bool) {
	c.Must(c.BindWithKey(key, initialize, prototype))
}

// Resolve inject args for func by callback
// callback func(...)
func (c *Container) Resolve(callback interface{}) error {
	_, err := c.Call(callback)
	return err
}

// MustResolve inject args for func by callback
func (c *Container) MustResolve(callback interface{}) {
	c.Must(c.Resolve(callback))
}

// ResolveWithError inject args for func by callback
// callback func(...) error
func (c *Container) ResolveWithError(callback interface{}) error {
	results, err := c.Call(callback)
	if err != nil {
		return newResolveError(err)
	}

	if len(results) == 1 && results[0] != nil {
		if err, ok := results[0].(error); ok {
			return err
		}
	}

	return nil
}

// CallWithProvider execute the callback with extra service provider
func (c *Container) CallWithProvider(callback interface{}, provider func() []*Entity) ([]interface{}, error) {
	callbackValue := reflect.ValueOf(callback)
	if !callbackValue.IsValid() {
		return nil, ErrInvalidArgs("callback is nil")
	}

	args, err := c.funcArgs(callbackValue.Type(), provider)
	if err != nil {
		return nil, err
	}

	returnValues := callbackValue.Call(args)
	results := make([]interface{}, len(returnValues))
	for index, val := range returnValues {
		results[index] = val.Interface()
	}

	return results, nil
}

// Call call a callback function and return it's results
func (c *Container) Call(callback interface{}) ([]interface{}, error) {
	return c.CallWithProvider(callback, func() []*Entity {
		return make([]*Entity, 0)
	})
}

// Get get instance by key from container
func (c *Container) Get(key interface{}) (interface{}, error) {
	return c.get(key, func() []*Entity {
		return make([]*Entity, 0)
	})
}

func (c *Container) get(key interface{}, provider func() []*Entity) (interface{}, error) {
	keyReflectType, ok := key.(reflect.Type)
	if !ok {
		keyReflectType = reflect.TypeOf(key)
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, obj := range provider() {

		if obj.key == key || obj.key == keyReflectType {
			return obj.Value(provider)
		}

		if obj.typ.AssignableTo(keyReflectType) {
			return obj.Value(provider)
		}
	}

	for _, obj := range c.objectSlices {

		if obj.key == key || obj.key == keyReflectType {
			return obj.Value(provider)
		}

		if obj.typ.AssignableTo(keyReflectType) {
			return obj.Value(provider)
		}
	}

	return nil, ErrObjectNotFound(fmt.Sprintf("key=%s", key))
}

// MustGet get instance by key from container
func (c *Container) MustGet(key interface{}) interface{} {
	res, err := c.Get(key)
	if err != nil {
		panic(err)
	}

	return res
}

func (c *Container) bindWith(key interface{}, typ reflect.Type, initialize interface{}, prototype bool) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.objects[key]; ok {
		return ErrRepeatedBind("key repeated")
	}

	entity := c.newEntity(key, typ, initialize, prototype)
	entity.index = len(c.objectSlices)

	c.objects[key] = entity
	c.objectSlices = append(c.objectSlices, entity)

	return nil
}

func (c *Container) funcArgs(t reflect.Type, provider func() []*Entity) ([]reflect.Value, error) {
	argsSize := t.NumIn()
	argValues := make([]reflect.Value, argsSize)
	for i := 0; i < argsSize; i++ {
		argType := t.In(i)
		val, err := c.instanceOfType(argType, provider)
		if err != nil {
			return argValues, err
		}

		argValues[i] = val
	}

	return argValues, nil
}

func (c *Container) instanceOfType(t reflect.Type, provider func() []*Entity) (reflect.Value, error) {
	arg, err := c.get(t, provider)
	if err != nil {
		return reflect.Value{}, ErrArgNotInstanced(err.Error())
	}

	return reflect.ValueOf(arg), nil
}

// Keys return all keys
func (c *Container) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	results := make([]interface{}, 0)
	for k := range c.objects {
		results = append(results, k)
	}

	return results
}

func isErrorType(t reflect.Type) bool {
	return t.Implements(reflect.TypeOf((*error)(nil)).Elem())
}

// ErrObjectNotFound is an error object represent object not found
func ErrObjectNotFound(msg string) error {
	return fmt.Errorf("the object can not be found in container: %s", msg)
}

// ErrArgNotInstanced is an error object represent arg not instanced
func ErrArgNotInstanced(msg string) error {
	return fmt.Errorf("the arg can not be found in container: %s", msg)
}

// ErrInvalidReturnValueCount is an error object represent return values count not match
func ErrInvalidReturnValueCount(msg string) error {
	return fmt.Errorf("invalid return value count: %s", msg)
}

// ErrRepeatedBind is an error object represent bind a value repeated
func ErrRepeatedBind(msg string) error {
	return fmt.Errorf("can not bind a value with repeated key: %s", msg)
}

// ErrInvalidArgs is an error object represent invalid args
func ErrInvalidArgs(msg string) error {
	return fmt.Errorf("invalid args: %s", msg)
}
