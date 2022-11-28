package ioc

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// container is a dependency injection container
type container struct {
	lock sync.RWMutex

	entities map[any]*Entity
	parent   Container
}

func (impl *container) P(initialize any) error {
	return impl.Prototype(initialize)
}

func (impl *container) S(initialize any) error {
	return impl.Singleton(initialize)
}

func (impl *container) V(key string, value any) error {
	return impl.BindValue(key, value)
}

func (impl *container) R(callback any) error {
	return impl.Resolve(callback)
}

func (impl *container) C(callback any) ([]any, error) {
	return impl.Call(callback)
}

func (impl *container) W(valPtr any) error {
	return impl.AutoWire(valPtr)
}

func (impl *container) MP(initialize any) {
	impl.MustPrototype(initialize)
}

func (impl *container) MS(initialize any) {
	impl.MustSingleton(initialize)
}

func (impl *container) MV(key string, value any) {
	impl.Must(impl.BindValue(key, value))
}

func (impl *container) MR(callback any) {
	impl.Must(impl.Resolve(callback))
}

func (impl *container) MW(valPtr any) {
	impl.MustAutoWire(valPtr)
}

func (impl *container) PrototypeOverride(initialize interface{}) error {
	return impl.Bind(initialize, true, true)
}

func (impl *container) MustPrototypeOverride(initialize interface{}) {
	impl.Must(impl.PrototypeOverride(initialize))
}

func (impl *container) PrototypeWithKeyOverride(key interface{}, initialize interface{}) error {
	return impl.BindWithKey(key, initialize, true, true)
}

func (impl *container) MustPrototypeWithKeyOverride(key interface{}, initialize interface{}) {
	impl.Must(impl.PrototypeWithKeyOverride(key, initialize))
}

func (impl *container) SingletonOverride(initialize interface{}) error {
	return impl.Bind(initialize, false, true)
}

func (impl *container) MustSingletonOverride(initialize interface{}) {
	impl.Must(impl.SingletonOverride(initialize))
}

func (impl *container) SingletonWithKeyOverride(key interface{}, initialize interface{}) error {
	return impl.BindWithKey(key, initialize, false, true)
}

func (impl *container) MustSingletonWithKeyOverride(key interface{}, initialize interface{}) {
	impl.Must(impl.SingletonWithKeyOverride(key, initialize))
}

// New create a new container
func New() Container {
	impl := &container{
		entities: make(map[any]*Entity),
	}

	impl.MustSingleton(func() Container { return impl })
	impl.MustSingleton(func() context.Context { return context.Background() })
	impl.MustSingleton(func() Binder { return impl })
	impl.MustSingleton(func() Resolver { return impl })

	return impl
}

// NewWithContext create a new container with context support
func NewWithContext(ctx context.Context) Container {
	cc := &container{
		entities: make(map[any]*Entity, 0),
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
	cc := &container{
		entities: make(map[any]*Entity, 0),
		parent:   c,
	}

	cc.MustSingleton(func() Container {
		return cc
	})

	return cc
}

// ExtendFrom extend from a parent container
func (impl *container) ExtendFrom(parent Container) {
	impl.parent = parent
}

// Must if err is not nil, panic it
func (impl *container) Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Prototype bind a prototype
// initialize func(...) (value, error)
func (impl *container) Prototype(initialize interface{}) error {
	return impl.Bind(initialize, true, false)
}

// MustPrototype bind a prototype, if failed then panic
func (impl *container) MustPrototype(initialize interface{}) {
	impl.Must(impl.Prototype(initialize))
}

// PrototypeWithKey bind a prototype with key
// initialize func(...) (value, error)
func (impl *container) PrototypeWithKey(key interface{}, initialize interface{}) error {
	return impl.BindWithKey(key, initialize, true, false)
}

// MustPrototypeWithKey bind a prototype with key, it failed, then panic
func (impl *container) MustPrototypeWithKey(key interface{}, initialize interface{}) {
	impl.Must(impl.PrototypeWithKey(key, initialize))
}

// Singleton bound a singleton
// initialize func(...) (value, error) or just a struct object
func (impl *container) Singleton(initialize interface{}) error {
	return impl.Bind(initialize, false, false)
}

// MustSingleton bind a singleton, if bind failed, then panic
func (impl *container) MustSingleton(initialize interface{}) {
	impl.Must(impl.Singleton(initialize))
}

// SingletonWithKey bind a singleton with key
// initialize func(...) (value, error)
func (impl *container) SingletonWithKey(key interface{}, initialize interface{}) error {
	return impl.BindWithKey(key, initialize, false, false)
}

// MustSingletonWithKey bind a singleton with key, if failed, then panic
func (impl *container) MustSingletonWithKey(key interface{}, initialize interface{}) {
	impl.Must(impl.SingletonWithKey(key, initialize))
}

// Provider create a provider from initializes
func (impl *container) Provider(initializes ...interface{}) EntitiesProvider {
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
func (impl *container) newEntityWrapper(initialize interface{}, prototype bool) (*Entity, error) {
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

func (impl *container) newEntity(key interface{}, typ reflect.Type, initialize interface{}, prototype bool, override bool) *Entity {
	entity := Entity{
		initializeFunc: initialize,
		key:            key,
		typ:            typ,
		value:          nil,
		c:              impl,
		prototype:      prototype,
		overridable:    override,
	}

	return &entity
}

func (impl *container) MustAutoWire(valPtr interface{}) {
	impl.Must(impl.AutoWire(valPtr))
}

func (impl *container) AutoWire(valPtr interface{}) error {
	if !reflect.ValueOf(valPtr).IsValid() {
		return buildInvalidArgsError("valPtr is nil")
	}

	valRef := reflect.ValueOf(valPtr)
	if valRef.Kind() != reflect.Ptr {
		return buildInvalidArgsError("valPtr must be a pointer to struct valPtr")
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
			val, err := impl.lookupInstance(tag, nil)
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
func (impl *container) Resolve(callback interface{}) error {
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

// MustResolve inject args for func by callback
func (impl *container) MustResolve(callback interface{}) {
	impl.Must(impl.Resolve(callback))
}

// CallWithProvider execute the callback with extra service provider
func (impl *container) CallWithProvider(callback interface{}, provider EntitiesProvider) ([]interface{}, error) {
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
func (impl *container) Call(callback interface{}) ([]interface{}, error) {
	return impl.CallWithProvider(callback, nil)
}

// Get instance by key from container
func (impl *container) Get(key interface{}) (interface{}, error) {
	return impl.lookupInstance(key, nil)
}

func (impl *container) lookupEntity(lookupKeys []any, provider func() []*Entity) *Entity {
	impl.lock.RLock()
	defer impl.lock.RUnlock()

	if provider != nil {
		for _, obj := range provider() {
			for _, lookupKey := range lookupKeys {
				if obj.key == lookupKey {
					return obj
				}
			}
		}
	}

	for _, lookupKey := range lookupKeys {
		if obj, ok := impl.entities[lookupKey]; ok {
			return obj
		}
	}

	return nil
}

func (impl *container) lookupInstance(key interface{}, provider func() []*Entity) (interface{}, error) {
	lookupKey, possibleKey := impl.resolveLookupKeys(key)
	obj := impl.lookupEntity(lookupKey, provider)
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

// resolveLookupKeys 解析用于查找的 Keys
// key 匹配规则为
//  1. matchKey == lookupKey ，则匹配
//  2. matchKey == type(lookupKey) ，则匹配
//  3. 如果 lookupKey 是指向接口的指针，则解析成接口本身，与 matchKey 比较，相等则匹配
func (impl *container) resolveLookupKeys(lookupKey interface{}) (lookupKeys []any, possibleKey any) {
	keyReflectType, lookupKeyIsReflectType := lookupKey.(reflect.Type)
	if !lookupKeyIsReflectType {
		keyReflectType = reflect.TypeOf(lookupKey)
	}

	lookupKeys = append(lookupKeys, lookupKey)
	if lookupKey != keyReflectType {
		lookupKeys = append(lookupKeys, keyReflectType)
	}

	switch keyReflectType.Kind() {
	case reflect.Ptr:
		typeUnderPointer := keyReflectType.Elem()
		switch typeUnderPointer.Kind() {
		case reflect.Interface:
			lookupKeys = append(lookupKeys, typeUnderPointer)
		default:
			possibleKey = typeUnderPointer
		}
	case reflect.Struct:
		if !lookupKeyIsReflectType {
			reflectValue := reflect.ValueOf(lookupKey)
			possibleKey = reflectValue.Addr().Type()
		}
	}

	return lookupKeys, possibleKey
}

// MustGet lookupInstance instance by key from container
func (impl *container) MustGet(key interface{}) interface{} {
	res, err := impl.Get(key)
	if err != nil {
		panic(err)
	}

	return res
}

func (impl *container) funcArgs(t reflect.Type, provider func() []*Entity) ([]reflect.Value, error) {
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

func (impl *container) instanceOfType(t reflect.Type, provider func() []*Entity) (reflect.Value, error) {
	arg, err := impl.lookupInstance(t, provider)
	if err != nil {
		return reflect.Value{}, buildArgNotInstancedError(err.Error())
	}

	return reflect.ValueOf(arg), nil
}

// Keys return all keys
func (impl *container) Keys() []interface{} {
	impl.lock.RLock()
	defer impl.lock.RUnlock()

	results := make([]any, 0, len(impl.entities))
	for k := range impl.entities {
		results = append(results, k)
	}

	return results
}

// CanOverride returns whether the key can be overridden
func (impl *container) CanOverride(key interface{}) (bool, error) {
	impl.lock.RLock()
	defer impl.lock.RUnlock()

	for _, obj := range impl.entities {
		if obj.key == key {
			return obj.overridable, nil
		}
	}

	return true, buildObjectNotFoundError(fmt.Sprintf("key=%#v not found", key))
}

// isValidKeyKind 判断类型是否允许作为key
func (impl *container) isValidKeyKind(kind reflect.Kind) error {
	if kind == reflect.Struct || kind == reflect.Interface || kind == reflect.Ptr {
		return nil
	}

	return buildInvalidArgsError(fmt.Sprintf("the type of key can not be a %v", kind))
}
