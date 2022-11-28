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

func (impl *containerImpl) PrototypeOverride(initialize interface{}) error {
	return impl.Bind(initialize, true, true)
}

func (impl *containerImpl) MustPrototypeOverride(initialize interface{}) {
	impl.Must(impl.PrototypeOverride(initialize))
}

func (impl *containerImpl) PrototypeWithKeyOverride(key interface{}, initialize interface{}) error {
	return impl.BindWithKey(key, initialize, true, true)
}

func (impl *containerImpl) MustPrototypeWithKeyOverride(key interface{}, initialize interface{}) {
	impl.Must(impl.PrototypeWithKeyOverride(key, initialize))
}

func (impl *containerImpl) SingletonOverride(initialize interface{}) error {
	return impl.Bind(initialize, false, true)
}

func (impl *containerImpl) MustSingletonOverride(initialize interface{}) {
	impl.Must(impl.SingletonOverride(initialize))
}

func (impl *containerImpl) SingletonWithKeyOverride(key interface{}, initialize interface{}) error {
	return impl.BindWithKey(key, initialize, false, true)
}

func (impl *containerImpl) MustSingletonWithKeyOverride(key interface{}, initialize interface{}) {
	impl.Must(impl.SingletonWithKeyOverride(key, initialize))
}

// New create a new container
func New() Container {
	impl := &containerImpl{
		objects:      make(map[interface{}]*Entity),
		objectSlices: make([]*Entity, 0),
	}

	impl.MustSingleton(func() Container { return impl })
	impl.MustSingleton(func() context.Context { return context.Background() })
	impl.MustSingleton(func() Binder { return impl })
	impl.MustSingleton(func() Resolver { return impl })

	return impl
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

// Extend create a new container, and it's parent is supplied container
// If it can not find a binding from current container, it will search from parents
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
func (impl *containerImpl) ExtendFrom(parent Container) {
	impl.parent = parent
}

// Must if err is not nil, panic it
func (impl *containerImpl) Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Prototype bind a prototype
// initialize func(...) (value, error)
func (impl *containerImpl) Prototype(initialize interface{}) error {
	return impl.Bind(initialize, true, false)
}

// MustPrototype bind a prototype, if failed then panic
func (impl *containerImpl) MustPrototype(initialize interface{}) {
	impl.Must(impl.Prototype(initialize))
}

// PrototypeWithKey bind a prototype with key
// initialize func(...) (value, error)
func (impl *containerImpl) PrototypeWithKey(key interface{}, initialize interface{}) error {
	return impl.BindWithKey(key, initialize, true, false)
}

// MustPrototypeWithKey bind a prototype with key, it failed, then panic
func (impl *containerImpl) MustPrototypeWithKey(key interface{}, initialize interface{}) {
	impl.Must(impl.PrototypeWithKey(key, initialize))
}

// Singleton bound a singleton
// initialize func(...) (value, error) or just a struct object
func (impl *containerImpl) Singleton(initialize interface{}) error {
	return impl.Bind(initialize, false, false)
}

// MustSingleton bind a singleton, if bind failed, then panic
func (impl *containerImpl) MustSingleton(initialize interface{}) {
	impl.Must(impl.Singleton(initialize))
}

// SingletonWithKey bind a singleton with key
// initialize func(...) (value, error)
func (impl *containerImpl) SingletonWithKey(key interface{}, initialize interface{}) error {
	return impl.BindWithKey(key, initialize, false, false)
}

// MustSingletonWithKey bind a singleton with key, if failed, then panic
func (impl *containerImpl) MustSingletonWithKey(key interface{}, initialize interface{}) {
	impl.Must(impl.SingletonWithKey(key, initialize))
}

// Provider create a provider from initializes
func (impl *containerImpl) Provider(initializes ...interface{}) EntitiesProvider {
	entities := make([]*Entity, len(initializes))
	for i, init := range initializes {
		entity, err := impl.newEntityWrapper(init, false)
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
func (impl *containerImpl) newEntityWrapper(initialize interface{}, prototype bool) (*Entity, error) {
	if !reflect.ValueOf(initialize).IsValid() {
		return nil, buildInvalidArgsError("initialize is nil")
	}

	initializeType := reflect.ValueOf(initialize).Type()
	if initializeType.NumOut() <= 0 {
		return nil, buildInvalidArgsError("expect func return values count greater than 0, but got 0")
	}

	typ := initializeType.Out(0)
	return impl.newEntity(typ, typ, initialize, prototype, true), nil
}

func (impl *containerImpl) newEntity(key interface{}, typ reflect.Type, initialize interface{}, prototype bool, override bool) *Entity {
	entity := Entity{
		initializeFunc: initialize,
		key:            key,
		typ:            typ,
		value:          nil,
		c:              impl,
		prototype:      prototype,
		override:       override,
	}

	return &entity
}

func (impl *containerImpl) MustAutoWire(object interface{}) {
	impl.Must(impl.AutoWire(object))
}

func (impl *containerImpl) AutoWire(object interface{}) error {
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
			val, err := impl.instanceOfType(field.Type, nil)
			if err != nil {
				return fmt.Errorf("%v: %v", field.Name, err)
			}

			fieldVal := structValue.Field(i)
			reflect.NewAt(fieldVal.Type(), unsafe.Pointer(fieldVal.UnsafeAddr())).Elem().Set(val)
		} else {
			val, err := impl.get(tag, nil)
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
func (impl *containerImpl) Resolve(callback interface{}) error {
	_, err := impl.Call(callback)
	return err
}

// MustResolve inject args for func by callback
func (impl *containerImpl) MustResolve(callback interface{}) {
	impl.Must(impl.Resolve(callback))
}

// ResolveWithError inject args for func by callback
// callback func(...) error
func (impl *containerImpl) ResolveWithError(callback interface{}) error {
	results, err := impl.Call(callback)
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
func (impl *containerImpl) CallWithProvider(callback interface{}, provider EntitiesProvider) ([]interface{}, error) {
	callbackValue, ok := callback.(reflect.Value)
	if !ok {
		callbackValue = reflect.ValueOf(callback)
	}

	if !callbackValue.IsValid() {
		return nil, buildInvalidArgsError("callback is nil")
	}

	args, err := impl.funcArgs(callbackValue.Type(), provider)
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

// Call a callback function and return its results
func (impl *containerImpl) Call(callback interface{}) ([]interface{}, error) {
	return impl.CallWithProvider(callback, nil)
}

// Get instance by key from container
func (impl *containerImpl) Get(key interface{}) (interface{}, error) {
	return impl.get(key, nil)
}

func (impl *containerImpl) getObj(lookupKey func(matchKey interface{}) bool, provider func() []*Entity) *Entity {
	impl.lock.RLock()
	defer impl.lock.RUnlock()

	if provider != nil {
		for _, obj := range provider() {
			if lookupKey(obj.key) {
				return obj
			}
		}
	}

	for _, obj := range impl.objectSlices {
		if lookupKey(obj.key) {
			return obj
		}
	}

	return nil
}

func (impl *containerImpl) get(key interface{}, provider func() []*Entity) (interface{}, error) {
	lookupKey, possibleKey := impl.buildKeyLookupFunc(key)
	obj := impl.getObj(lookupKey, provider)
	if obj != nil {
		return obj.Value(provider)
	}

	if impl.parent != nil {
		return impl.parent.Get(key)
	}

	errMsg := fmt.Sprintf("key=%v not found", key)
	if possibleKey != nil {
		errMsg = fmt.Sprintf("%s, may be you want %v", errMsg, possibleKey)
	}
	return nil, buildObjectNotFoundError(errMsg)
}

// buildKeyLookupFunc 构建用于查询 key 是否存在的函数
// key 匹配规则为
//  1. matchKey == lookupKey ，则匹配
//  2. matchKey == type(lookupKey) ，则匹配
//  3. 如果 lookupKey 是指向接口的指针，则解析成接口本身，与 matchKey 比较，相等则匹配
func (impl *containerImpl) buildKeyLookupFunc(lookupKey interface{}) (lookupFunc func(matchKey interface{}) bool, possibleKey interface{}) {
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
func (impl *containerImpl) MustGet(key interface{}) interface{} {
	res, err := impl.Get(key)
	if err != nil {
		panic(err)
	}

	return res
}

func (impl *containerImpl) funcArgs(t reflect.Type, provider func() []*Entity) ([]reflect.Value, error) {
	argsSize := t.NumIn()
	argValues := make([]reflect.Value, argsSize)
	for i := 0; i < argsSize; i++ {
		argType := t.In(i)
		val, err := impl.instanceOfType(argType, provider)
		if err != nil {
			return argValues, err
		}

		argValues[i] = val
	}

	return argValues, nil
}

func (impl *containerImpl) instanceOfType(t reflect.Type, provider func() []*Entity) (reflect.Value, error) {
	arg, err := impl.get(t, provider)
	if err != nil {
		return reflect.Value{}, buildArgNotInstancedError(err.Error())
	}

	return reflect.ValueOf(arg), nil
}

// Keys return all keys
func (impl *containerImpl) Keys() []interface{} {
	impl.lock.RLock()
	defer impl.lock.RUnlock()

	results := make([]interface{}, 0)
	for _, k := range impl.objectSlices {
		results = append(results, k)
	}

	return results
}

// CanOverride returns whether the key can be override
func (impl *containerImpl) CanOverride(key interface{}) (bool, error) {
	impl.lock.RLock()
	defer impl.lock.RUnlock()

	obj, ok := impl.objects[key]
	if !ok {
		return true, buildObjectNotFoundError(fmt.Sprintf("key=%#v not found", key))
	}

	return obj.override, nil
}

// isValidKeyKind 判断类型是否允许作为key
func (impl *containerImpl) isValidKeyKind(kind reflect.Kind) error {
	if kind == reflect.Struct || kind == reflect.Interface || kind == reflect.Ptr {
		return nil
	}

	return buildInvalidArgsError(fmt.Sprintf("the type of key can not be a %v", kind))
}
