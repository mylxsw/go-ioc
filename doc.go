/*
Package container 实现了依赖注入容器，用于管理Golang对象的创建。

	c := container.New()

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
package container

type Container interface {
	Prototype(initialize interface{}) error
	MustPrototype(initialize interface{})
	PrototypeWithKey(key interface{}, initialize interface{}) error
	MustPrototypeWithKey(key interface{}, initialize interface{})

	PrototypeOverride(initialize interface{}) error
	MustPrototypeOverride(initialize interface{})
	PrototypeWithKeyOverride(key interface{}, initialize interface{}) error
	MustPrototypeWithKeyOverride(key interface{}, initialize interface{})

	Singleton(initialize interface{}) error
	MustSingleton(initialize interface{})
	SingletonWithKey(key interface{}, initialize interface{}) error
	MustSingletonWithKey(key interface{}, initialize interface{})

	SingletonOverride(initialize interface{}) error
	MustSingletonOverride(initialize interface{})
	SingletonWithKeyOverride(key interface{}, initialize interface{}) error
	MustSingletonWithKeyOverride(key interface{}, initialize interface{})

	HasBoundValue(key string) bool
	BindValue(key string, value interface{}) error
	MustBindValue(key string, value interface{})
	BindValueOverride(key string, value interface{}) error
	MustBindValueOverride(key string, value interface{})

	HasBound(key interface{}) bool
	Bind(initialize interface{}, prototype bool, override bool) error
	MustBind(initialize interface{}, prototype bool, override bool)
	BindWithKey(key interface{}, initialize interface{}, prototype bool, override bool) error
	MustBindWithKey(key interface{}, initialize interface{}, prototype bool, override bool)

	Resolve(callback interface{}) error
	MustResolve(callback interface{})
	ResolveWithError(callback interface{}) error
	CallWithProvider(callback interface{}, provider func() []*Entity) ([]interface{}, error)
	Call(callback interface{}) ([]interface{}, error)
	// AutoWire 自动对结构体对象进行依赖注入，object 必须是结构体对象的指针
	// 自动注入字段（公开和私有均支持）需要添加 `autowire` tag，支持以下两种
	//  - autowire:"@" 根据字段的类型来注入
	//  - autowire:"自定义key" 根据自定义的key来注入（查找名为 key 的绑定）
	AutoWire(object interface{}) error

	Get(key interface{}) (interface{}, error)
	MustGet(key interface{}) interface{}

	Provider(initializes ...interface{}) (func() []*Entity, error)
	ExtendFrom(parent Container)
	Must(err error)
	Keys() []interface{}
	CanOverride(key interface{}) (bool, error)
}
