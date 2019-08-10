/*
Package container 实现了依赖注入容器，用于管理Golang对象的创建。

	c := container.New()

	c.BindValue("conn_str", "root:root@/my_db?charset=utf8")
	c.Singleton(func(c *container.Container) (*UserRepo, error) {
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
