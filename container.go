package container

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// Entity represent a entity in container
type Entity struct {
	lock sync.RWMutex

	key            interface{}  // entity key
	initializeFunc interface{}  // initializeFunc is a func to initialize entity
	value          interface{}  // the value of initializeFunc
	typ            reflect.Type // the type of value
	index          int          // the index in the container
	override       bool         // identify whether the entity can be override

	prototype bool
	c         *containerImpl
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

// containerImpl is a dependency injection container
type containerImpl struct {
	lock sync.RWMutex

	objects      map[interface{}]*Entity
	objectSlices []*Entity

	parent Container
}

func (c *containerImpl) PrototypeOverride(initialize interface{}) error {
	return c.Bind(initialize, true, true)
}

func (c *containerImpl) MustPrototypeOverride(initialize interface{}) {
	c.Must(c.PrototypeOverride(initialize))
}

func (c *containerImpl) PrototypeWithKeyOverride(key interface{}, initialize interface{}) error {
	return c.BindWithKey(key, initialize, true, true)
}

func (c *containerImpl) MustPrototypeWithKeyOverride(key interface{}, initialize interface{}) {
	c.Must(c.PrototypeWithKeyOverride(key, initialize))
}

func (c *containerImpl) SingletonOverride(initialize interface{}) error {
	return c.Bind(initialize, false, true)
}

func (c *containerImpl) MustSingletonOverride(initialize interface{}) {
	c.Must(c.SingletonOverride(initialize))
}

func (c *containerImpl) SingletonWithKeyOverride(key interface{}, initialize interface{}) error {
	return c.BindWithKey(key, initialize, false, true)
}

func (c *containerImpl) MustSingletonWithKeyOverride(key interface{}, initialize interface{}) {
	c.Must(c.SingletonWithKeyOverride(key, initialize))
}

// New create a new container
func New() Container {
	cc := &containerImpl{
		objects:      make(map[interface{}]*Entity),
		objectSlices: make([]*Entity, 0),
	}

	cc.MustSingleton(func() Container { return cc })
	cc.MustSingleton(func() context.Context { return context.Background() })
	cc.MustSingleton(func() Binder { return cc })
	cc.MustSingleton(func() Resolver { return cc })

	return cc
}

// NewWithContext create a new container with context support
func NewWithContext(ctx context.Context) Container {
	cc := &containerImpl{
		objects:      make(map[interface{}]*Entity),
		objectSlices: make([]*Entity, 0),
	}

	cc.MustSingleton(func() Container { return cc })
	cc.MustSingleton(func() context.Context { return ctx })
	cc.MustSingleton(func() Binder { return cc })
	cc.MustSingleton(func() Resolver { return cc })

	return cc
}

// Extend create a new container and it's parent is supplied container
// If can not found a binding from current container, it will search from parents
func Extend(c Container) Container {
	cc := &containerImpl{
		objects:      make(map[interface{}]*Entity),
		objectSlices: make([]*Entity, 0),
		parent:       c,
	}

	cc.MustSingleton(func() Container {
		return cc
	})

	return cc
}

// ExtendFrom extend from a parent containerImpl
func (c *containerImpl) ExtendFrom(parent Container) {
	c.parent = parent
}

// Must if err is not nil, panic it
func (c *containerImpl) Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Prototype bind a prototype
// initialize func(...) (value, error)
func (c *containerImpl) Prototype(initialize interface{}) error {
	return c.Bind(initialize, true, false)
}

// MustPrototype bind a prototype, if failed then panic
func (c *containerImpl) MustPrototype(initialize interface{}) {
	c.Must(c.Prototype(initialize))
}

// PrototypeWithKey bind a prototype with key
// initialize func(...) (value, error)
func (c *containerImpl) PrototypeWithKey(key interface{}, initialize interface{}) error {
	return c.BindWithKey(key, initialize, true, false)
}

// MustPrototypeWithKey bind a prototype with key, it failed, then panic
func (c *containerImpl) MustPrototypeWithKey(key interface{}, initialize interface{}) {
	c.Must(c.PrototypeWithKey(key, initialize))
}

// Singleton bind a singleton
// initialize func(...) (value, error) or just an struct object
func (c *containerImpl) Singleton(initialize interface{}) error {
	return c.Bind(initialize, false, false)
}

// MustSingleton bind a singleton, if bind failed, then panic
func (c *containerImpl) MustSingleton(initialize interface{}) {
	c.Must(c.Singleton(initialize))
}

// SingletonWithKey bind a singleton with key
// initialize func(...) (value, error)
func (c *containerImpl) SingletonWithKey(key interface{}, initialize interface{}) error {
	return c.BindWithKey(key, initialize, false, false)
}

// MustSingletonWithKey bind a singleton with key, if failed, then panic
func (c *containerImpl) MustSingletonWithKey(key interface{}, initialize interface{}) {
	c.Must(c.SingletonWithKey(key, initialize))
}

// Provider create a provider from initializes
func (c *containerImpl) Provider(initializes ...interface{}) EntitiesProvider {
	entities := make([]*Entity, len(initializes))
	for i, init := range initializes {
		entity, err := c.newEntityWrapper(init, false)
		if err != nil {
			panic(err)
		}

		entities[i] = entity
	}

	return func() []*Entity {
		return entities
	}
}

// newEntityWrapper create a new entity
func (c *containerImpl) newEntityWrapper(initialize interface{}, prototype bool) (*Entity, error) {
	if !reflect.ValueOf(initialize).IsValid() {
		return nil, buildInvalidArgsError("initialize is nil")
	}

	initializeType := reflect.ValueOf(initialize).Type()
	if initializeType.NumOut() <= 0 {
		return nil, buildInvalidArgsError("expect func return values count greater than 0, but got 0")
	}

	typ := initializeType.Out(0)
	return c.newEntity(typ, typ, initialize, prototype, true), nil
}

func (c *containerImpl) newEntity(key interface{}, typ reflect.Type, initialize interface{}, prototype bool, override bool) *Entity {
	entity := Entity{
		initializeFunc: initialize,
		key:            key,
		typ:            typ,
		value:          nil,
		c:              c,
		prototype:      prototype,
		override:       override,
	}

	return &entity
}

func (c *containerImpl) MustAutoWire(object interface{}) {
	c.Must(c.AutoWire(object))
}

func (c *containerImpl) AutoWire(object interface{}) error {
	if !reflect.ValueOf(object).IsValid() {
		return buildInvalidArgsError("object is nil")
	}

	valRef := reflect.ValueOf(object)
	if valRef.Kind() != reflect.Ptr {
		return buildInvalidArgsError("object must be a pointer to struct object")
	}

	structValue := valRef.Elem()
	structType := structValue.Type()
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("autowire")
		if tag == "" || tag == "-" {
			continue
		}

		if tag == "@" {
			val, err := c.instanceOfType(field.Type, nil)
			if err != nil {
				return fmt.Errorf("%v: %v", field.Name, err)
			}

			fieldVal := structValue.Field(i)
			reflect.NewAt(fieldVal.Type(), unsafe.Pointer(fieldVal.UnsafeAddr())).Elem().Set(val)
		} else {
			val, err := c.get(tag, nil)
			if err != nil {
				return fmt.Errorf("%v: %v", field.Name, err)
			}

			fieldVal := structValue.Field(i)
			reflect.NewAt(fieldVal.Type(), unsafe.Pointer(fieldVal.UnsafeAddr())).Elem().Set(reflect.ValueOf(val))
		}
	}

	return nil
}

// Resolve inject args for func by callback
// callback func(...)
func (c *containerImpl) Resolve(callback interface{}) error {
	_, err := c.Call(callback)
	return err
}

// MustResolve inject args for func by callback
func (c *containerImpl) MustResolve(callback interface{}) {
	c.Must(c.Resolve(callback))
}

// ResolveWithError inject args for func by callback
// callback func(...) error
func (c *containerImpl) ResolveWithError(callback interface{}) error {
	results, err := c.Call(callback)
	if err != nil {
		return err
	}

	if len(results) == 1 && results[0] != nil {
		if err, ok := results[0].(error); ok {
			return err
		}
	}

	return nil
}

// CallWithProvider execute the callback with extra service provider
func (c *containerImpl) CallWithProvider(callback interface{}, provider EntitiesProvider) ([]interface{}, error) {
	callbackValue := reflect.ValueOf(callback)
	if !callbackValue.IsValid() {
		return nil, buildInvalidArgsError("callback is nil")
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
func (c *containerImpl) Call(callback interface{}) ([]interface{}, error) {
	return c.CallWithProvider(callback, nil)
}

// Get get instance by key from container
func (c *containerImpl) Get(key interface{}) (interface{}, error) {
	return c.get(key, nil)
}

func (c *containerImpl) getObj(lookupKey func(matchKey interface{}) bool, provider func() []*Entity) *Entity {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if provider != nil {
		for _, obj := range provider() {
			if lookupKey(obj.key) {
				return obj
			}
		}
	}

	for _, obj := range c.objectSlices {
		if lookupKey(obj.key) {
			return obj
		}
	}

	return nil
}

func (c *containerImpl) get(key interface{}, provider func() []*Entity) (interface{}, error) {
	lookupKey, possibleKey := c.buildKeyLookupFunc(key)
	obj := c.getObj(lookupKey, provider)
	if obj != nil {
		return obj.Value(provider)
	}

	if c.parent != nil {
		return c.parent.Get(key)
	}

	errMsg := fmt.Sprintf("key=%v not found", key)
	if possibleKey != nil {
		errMsg = fmt.Sprintf("%s, may be you want %v", errMsg, possibleKey)
	}
	return nil, buildObjectNotFoundError(errMsg)
}

// buildKeyLookupFunc 构建用于查询 key 是否存在的函数
// key 匹配规则为
//    1. matchKey == lookupKey ，则匹配
//    2. matchKey == type(lookupKey) ，则匹配
//    3. 如果 lookupKey 是指向接口的指针，则解析成接口本身，与 matchKey 比较，相等则匹配
func (c *containerImpl) buildKeyLookupFunc(lookupKey interface{}) (lookupFunc func(matchKey interface{}) bool, possibleKey interface{}) {
	keyReflectType, lookupKeyIsReflectType := lookupKey.(reflect.Type)
	if !lookupKeyIsReflectType {
		keyReflectType = reflect.TypeOf(lookupKey)
	}

	keyLookupMap := make(map[interface{}]bool)
	keyLookupMap[lookupKey] = true
	if lookupKey != keyReflectType {
		keyLookupMap[keyReflectType] = true
	}

	switch keyReflectType.Kind() {
	case reflect.Ptr:
		typeUnderPointer := keyReflectType.Elem()
		switch typeUnderPointer.Kind() {
		case reflect.Interface:
			keyLookupMap[typeUnderPointer] = true
		default:
			possibleKey = typeUnderPointer
		}
	case reflect.Struct:
		if !lookupKeyIsReflectType {
			reflectValue := reflect.ValueOf(lookupKey)
			possibleKey = reflectValue.Addr().Type()
		}
	}

	return func(key interface{}) bool {
		_, ok := keyLookupMap[key]
		return ok
	}, possibleKey
}

// MustGet get instance by key from container
func (c *containerImpl) MustGet(key interface{}) interface{} {
	res, err := c.Get(key)
	if err != nil {
		panic(err)
	}

	return res
}

func (c *containerImpl) funcArgs(t reflect.Type, provider func() []*Entity) ([]reflect.Value, error) {
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

func (c *containerImpl) instanceOfType(t reflect.Type, provider func() []*Entity) (reflect.Value, error) {
	arg, err := c.get(t, provider)
	if err != nil {
		return reflect.Value{}, buildArgNotInstancedError(err.Error())
	}

	return reflect.ValueOf(arg), nil
}

// Keys return all keys
func (c *containerImpl) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	results := make([]interface{}, 0)
	for k := range c.objects {
		results = append(results, k)
	}

	return results
}

// CanOverride returns whether the key can be override
func (c *containerImpl) CanOverride(key interface{}) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	obj, ok := c.objects[key]
	if !ok {
		return true, buildObjectNotFoundError(fmt.Sprintf("key=%#v not found", key))
	}

	return obj.override, nil
}

// isValidKeyKind 判断类型是否允许作为key
func (c *containerImpl) isValidKeyKind(kind reflect.Kind) error {
	if kind == reflect.Struct || kind == reflect.Interface || kind == reflect.Ptr {
		return nil
	}

	return buildInvalidArgsError(fmt.Sprintf("the type of key can not be a %v", kind))
}
