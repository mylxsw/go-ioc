/*
Package ioc 实现了依赖注入容器，用于管理Golang对象的创建。

	c := ioc.New()

	c.BindValue("conn_str", "root:root@/my_db?charset=utf8")
	c.Singleton(func(c container.Container) (*UserRepo, error) {
		connStr, err := c.Get("conn_str")
		if err != nil {
			return nil, err
		}

		return &UserRepo{connStr: connStr.(string)}, nil
	})
	c.Prototype(func(userRepo *UserRepo) *UserService {
		return &UserService{repo: userRepo}
	})

	if err := c.Resolve(func(userService *UserService) {
		if userService.GetUser() != expectedValue {
			t.Error("test failed")
		}
	}); err != nil {
		t.Errorf("test failed: %s", err)
		return
	}
*/
package ioc

type Container interface {
	// P alias of Prototype
	P(initialize any) error
	// S alias of Singleton
	S(initialize any) error
	// V alias of BindValue
	V(key string, value any) error
	// R alias of Resolve
	R(callback any) error
	// C alias of Call
	C(callback any) ([]any, error)
	// W alias of AutoWire
	W(valPtr any) error

	// MP alias of MustPrototype
	MP(initialize any)
	// MS alias of MustSingleton
	MS(initialize any)
	// MV alias of MustBindValue
	MV(key string, value any)
	// MR alias of MustResolve
	MR(callback any)
	// MW alias of MustAutoWire
	MW(valPtr any)

	Prototype(initialize any) error
	MustPrototype(initialize any)
	PrototypeWithKey(key any, initialize any) error
	MustPrototypeWithKey(key any, initialize any)

	PrototypeOverride(initialize any) error
	MustPrototypeOverride(initialize any)
	PrototypeWithKeyOverride(key any, initialize any) error
	MustPrototypeWithKeyOverride(key any, initialize any)

	Singleton(initialize any) error
	MustSingleton(initialize any)
	SingletonWithKey(key any, initialize any) error
	MustSingletonWithKey(key any, initialize any)

	SingletonOverride(initialize any) error
	MustSingletonOverride(initialize any)
	SingletonWithKeyOverride(key any, initialize any) error
	MustSingletonWithKeyOverride(key any, initialize any)

	BindValue(key string, value any) error
	MustBindValue(key string, value any)
	BindValueOverride(key string, value any) error
	MustBindValueOverride(key string, value any)

	Bind(initialize any, prototype bool, override bool) error
	MustBind(initialize any, prototype bool, override bool)
	BindWithKey(key any, initialize any, prototype bool, override bool) error
	MustBindWithKey(key any, initialize any, prototype bool, override bool)

	Resolve(callback any) error
	MustResolve(callback any)
	CallWithProvider(callback any, provider EntitiesProvider) ([]any, error)
	Call(callback any) ([]any, error)
	// AutoWire 自动对结构体对象进行依赖注入，insPtr 必须是结构体对象的指针
	// 自动注入字段（公开和私有均支持）需要添加 `autowire` tag，支持以下两种
	//  - autowire:"@" 根据字段的类型来注入
	//  - autowire:"自定义key" 根据自定义的key来注入（查找名为 key 的绑定）
	AutoWire(insPtr any) error
	MustAutoWire(insPtr any)

	Get(key any) (any, error)
	MustGet(key any) any

	Provider(initializes ...any) EntitiesProvider
	ExtendFrom(parent Container)

	Must(err error)
	Keys() []any
	CanOverride(key any) (bool, error)
	HasBoundValue(key string) bool
	HasBound(key any) bool
}

type Binder interface {
	// P alias of Prototype
	P(initialize any) error
	// S alias of Singleton
	S(initialize any) error
	// V alias of BindValue
	V(key string, value any) error

	// MP alias of MustPrototype
	MP(initialize any)
	// MS alias of MustSingleton
	MS(initialize any)
	// MV alias of MustBindValue
	MV(key string, value any)

	Prototype(initialize any) error
	MustPrototype(initialize any)
	PrototypeWithKey(key any, initialize any) error
	MustPrototypeWithKey(key any, initialize any)

	PrototypeOverride(initialize any) error
	MustPrototypeOverride(initialize any)
	PrototypeWithKeyOverride(key any, initialize any) error
	MustPrototypeWithKeyOverride(key any, initialize any)

	Singleton(initialize any) error
	MustSingleton(initialize any)
	SingletonWithKey(key any, initialize any) error
	MustSingletonWithKey(key any, initialize any)

	SingletonOverride(initialize any) error
	MustSingletonOverride(initialize any)
	SingletonWithKeyOverride(key any, initialize any) error
	MustSingletonWithKeyOverride(key any, initialize any)

	BindValue(key string, value any) error
	MustBindValue(key string, value any)
	BindValueOverride(key string, value any) error
	MustBindValueOverride(key string, value any)

	Bind(initialize any, prototype bool, override bool) error
	MustBind(initialize any, prototype bool, override bool)
	BindWithKey(key any, initialize any, prototype bool, override bool) error
	MustBindWithKey(key any, initialize any, prototype bool, override bool)

	Must(err error)
	Keys() []any
	CanOverride(key any) (bool, error)
	HasBoundValue(key string) bool
	HasBound(key any) bool
}

type EntitiesProvider func() []*Entity

type Resolver interface {
	// R alias of Resolve
	R(callback any) error
	// C alias of Call
	C(callback any) ([]any, error)
	// W alias of AutoWire
	W(valPtr any) error
	// MR alias of MustResolve
	MR(callback any)
	// MW alias of MustAutoWire
	MW(valPtr any)

	Resolve(callback any) error
	MustResolve(callback any)
	CallWithProvider(callback any, provider EntitiesProvider) ([]any, error)
	Provider(initializes ...any) EntitiesProvider
	Call(callback any) ([]any, error)
	// AutoWire 自动对结构体对象进行依赖注入，object 必须是结构体对象的指针
	// 自动注入字段（公开和私有均支持）需要添加 `autowire` tag，支持以下两种
	//  - autowire:"@" 根据字段的类型来注入
	//  - autowire:"自定义key" 根据自定义的key来注入（查找名为 key 的绑定）
	AutoWire(object any) error
	MustAutoWire(object any)

	Get(key any) (any, error)
	MustGet(key any) any

	Must(err error)
	Keys() []any
	HasBoundValue(key string) bool
	HasBound(key any) bool
}
