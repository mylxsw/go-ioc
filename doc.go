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
	ExtendFrom(parent Container)
	Must(err error)
	Prototype(initialize interface{}) error
	MustPrototype(initialize interface{})
	PrototypeWithKey(key interface{}, initialize interface{}) error
	MustPrototypeWithKey(key interface{}, initialize interface{})
	Singleton(initialize interface{}) error
	MustSingleton(initialize interface{})
	SingletonWithKey(key interface{}, initialize interface{}) error
	MustSingletonWithKey(key interface{}, initialize interface{})
	BindValue(key interface{}, value interface{}) error
	MustBindValue(key interface{}, value interface{})
	ServiceProvider(initializes ...interface{}) (func() []*Entity, error)
	NewEntity(initialize interface{}, prototype bool) (*Entity, error)
	Bind(initialize interface{}, prototype bool) error
	MustBind(initialize interface{}, prototype bool)
	BindWithKey(key interface{}, initialize interface{}, prototype bool) error
	MustBindWithKey(key interface{}, initialize interface{}, prototype bool)
	Resolve(callback interface{}) error
	MustResolve(callback interface{})
	ResolveWithError(callback interface{}) error
	CallWithProvider(callback interface{}, provider func() []*Entity) ([]interface{}, error)
	Call(callback interface{}) ([]interface{}, error)
	Get(key interface{}) (interface{}, error)
	MustGet(key interface{}) interface{}
	Keys() []interface{}
}
