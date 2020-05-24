# Container

[![Build Status](https://www.travis-ci.org/mylxsw/container.svg?branch=master)](https://www.travis-ci.org/mylxsw/container)
[![Coverage Status](https://coveralls.io/repos/github/mylxsw/container/badge.svg?branch=master)](https://coveralls.io/github/mylxsw/container?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/mylxsw/container)](https://goreportcard.com/report/github.com/mylxsw/container)
[![codecov](https://codecov.io/gh/mylxsw/container/branch/master/graph/badge.svg)](https://codecov.io/gh/mylxsw/container)
[![Sourcegraph](https://sourcegraph.com/github.com/mylxsw/container/-/badge.svg)](https://sourcegraph.com/github.com/mylxsw/container?badge)
[![GitHub](https://img.shields.io/github/license/mylxsw/container.svg)](https://github.com/mylxsw/container)


**Container** 是一款为 Go 语言开发的运行时依赖注入库。Go 语言的语言特性决定了实现一款类型安全的依赖注入容器并不太容易，因此 **Container** 大量使用了 Go 的反射机制。如果你的使用场景对性能要求并不是那个苛刻，那么建议你尝试一下它。

> 并不是说对性能要求苛刻的环境中就不能使用了，你可以把 **Container** 作为一个对象依赖管理工具，在你的业务初始化时获取依赖的对象。

要创建一个 **Container** 实例，使用 `containier.New` 方法

	cc := container.New()

此时就创建了一个空的容器。

> 你也可以使用 `container.NewWithContext(ctx)` 来创建容器，创建之后，可以自动的把已经存在的 `context.Context` 对象添加到容器中，由容器托管。

## 对象绑定

在使用之前，我们需要先将我们要托管的对象告诉容器。**Container** 支持三种类型的对象管理

- 单例对象 `Singleton`
- 原型对象（多例对象） `Prototype`
- 字符串值对象绑定 `Value`

> 所以的对象绑定方法都会返回一个 `error` 返回值来说明是否绑定成功，应用在使用时一定要主动去检查这个 `error`。
> 
> 确定对象一定会绑定成功（一般不违反文档中描述的参数签名方式，都是一定会成功的）或者要求对象必须要绑定成功（通常我们都要求这样，不然怎么进行依赖管理呢），则可以使用 `Must` 系列方法，比如 `Singleton` 方法对应的时 `MustSingleton`，当创建出错时，该方法会直接 `panic`。

绑定对象时，`Singleton`，`Prototype`，`BindValue` 方法对于同一类型，只能绑定一次，如果多次绑定同一类型对象的创建函数，会返回 `ErrRepeatedBind` 错误。

有时候，希望对象创建函数可以多次重新绑定，这样就可以个应用更多的扩展性，可以随时替换掉对象的创建方法，比如测试时 `Mock` 对象的注入。这时候我们可以使用 `Override` 系列方法：

- `SingletonOverride`
- `PrototypeOverride`
- `BindValueOverride`

使用 `Override` 系列方法时，必须保证第一次绑定时使用的是 `Override` 系列方法，否则无法重新绑定。

> 也就是说，可以这样绑定 `SingletonOverride` -> `SingletonOverride` ，`SingletonOverride` -> `Singleton`，但是一旦出现 `Singleton`，后续就无法对该对象重新绑定了。



### 单例对象

使用 `Singleton` 系列的方法来将单例对象托管给容器，**单例对象只会在第一次使用时自动完成创建**，之后所有对该对象的访问都会自动将已经创建好的对象注入进来。

常用的方法是 `Singleton(initialize interface{}) error` 方法，该方法会按照你提供的 `initialize` 函数或者对象来完成单例对象的注册。

参数 `initialize` 支持以下几种形式：

- 对象创建函数 `func(deps...) 对象返回值` 

	比如 

		cc.Singleton(func() UserRepo { return &userRepoImpl{} })
		cc.Singleton(func() (*sql.DB, error) {
			return sql.Open("mysql", "user:pwd@tcp(ip:3306)/dbname")
		})
		cc.Singleton(func(db *sql.DB) UserRepo { 
			// 这里我们创建的 userRepoImpl 对象，依赖 sql.DB 对象，只需要在函数
			// 参数中，将依赖列举出来，容器会自动完成这些对象的创建
			return &userRepoImpl{db: db} 
		})

- 带错误返回值的对象创建函数 `func(deps...) (对象返回值, error)`

	对象创建函数最多支持两个返回值，且要求第一个返回值为期望创建的对象，第二个返回值为 error 对象。

		cc.Singleton(func() (Config, error) {
			// 假设我们要创建配置对象，该对象的初始化时从文件读取配置
			content, err := ioutil.ReadFile("test.conf")
			if err != nil {
				return nil, err
			}

			return config.Load(content), nil
		})

- 直接绑定对象 

	如果对象已经创建好了，想要让 **Container** 来管理，可以直接将对象传递 `Singleton` 方法

		userRepo := repo.NewUserRepo()
		cc.Singleton(userRepo)


> 当对象第一次被使用时，**Container** 会将对象创建函数的执行结果缓存起来，从而实现任何时候后访问都是获取到的同一个对象。

### 原型对象（多例对象）

原型对象（多例对象）是指的由 **Container** 托管对象的创建过程，但是每次使用依赖注入获取到的都是新创建的对象。

使用 `Prototype` 系列的方法来将原型对象的创建托管给容器。常用的方法是 `Prototype(initialize interface{}) error`。

参数 `initialize` 可以接受的类型与 `Singleton` 系列函数完全一致，唯一的区别是在对象使用时，单例对象每次都是返回的同一个对象，而原型对象则是每次都返回新创建的对象。

### 字符串值对象绑定

这种绑定方式是将某个对象绑定到 **Container** 中，但是与 `Singleton` 系列方法不同的是，它要求必须指定一个字符串类型的 `Key`，每次获取对象的时候，使用 `Get` 系列函数获取绑定的对象时，直接传递这个字符串 Key 即可。

常用的绑定方法为 `BindValue(key string, value interface{})`。

	cc.BindValue("version", "1.0.1")
	cc.MustBindValue("startTs", time.Now())
	cc.BindValue("int_val", 123)


## 依赖注入

在使用绑定对象时，通常我们使用 `Resolve` 和 `Call` 系列方法。

### Resolve

`Resolve(callback interface{}) error` 方法执行体 callback 内部只能进行依赖注入，不接收注入函数的返回值，虽然有一个 `error` 返回值，但是该值只表明是否在注入对象时产生错误。

比如，我们需要获取某个用户的信息和其角色信息，使用 Resolve 方法

	cc.MustResolve(func(userRepo repo.UserRepo, roleRepo repo.RoleRepo) {
		// 查询 id=123 的用户，查询失败直接panic
		user, err := userRepo.GetUser(123)
		if err != nil {
			panic(err)
		}
		// 查询用户角色，查询失败时，我们忽略了返回的错误
		role, _ := roleRepo.GetRole(user.RoleID)

		// do something you want with user/role
	})

直接使用 `Resolve` 方法可能并不太满足我们的日常业务需求，因为在执行查询的时候，总是会遇到各种 `error`，直接丢弃会产生很多隐藏的 Bug，但是我们也不倾向于使用 `Panic` 这种暴力的方式来解决。

**Container** 提供了 `ResolveWithError(callback interface{}) error` 方法，使用该方法时，我们的 callback 可以接受一个 `error` 返回值，来告诉调用者这里出现问题了。

	err := cc.ResolveWithError(func(userRepo repo.UserRepo, roleRepo repo.RoleRepoo) error {
		user, err := userRepo.GetUser(123)
		if err != nil {
			return err
		}

		role, err := roleRepo.GetRole(user.RoleID)
		if err != nil {
			return err
		}

		// do something you want with user/role

		return nil
	})
	if err != nil {
		// 自定义错误处理
	}


### Call


`Call(callback interface{}) ([]interface{}, error)` 方法不仅完成对象的依赖注入，还会返回 `callback` 的返回值，返回值为数组结构。

比如

	results, err := cc.Call(func(userRepo repo.UserRepo) ([]repo.User, error) {
		users, err := userRepo.AllUsers()
		return users, err
	})
	if err != nil {
		// 这里的 err 是依赖注入过程中的错误，比如依赖对象创建失败
	}

	// results 是一个类型为 []interface{} 的数组，数组中按次序包含了 callback 函数的返回值
	// results[0] - []repo.User
	// results[1] - error
	// 由于每个返回值都是 interface{} 类型，因此在使用时需要执行类型断言，将其转换为具体的类型再使用
	users := results[0].([]repo.User)
	err := results[0].(error)


### Provider 

有时我们希望为不同的功能模块绑定不同的对象实现，比如在 Web 服务器中，每个请求的 handler 函数需要访问与本次请求有关的 request/response 对象，请求结束之后，**Container** 中的 request/response 对象也就没有用了，不同的请求获取到的也不是同一个对象。我们可以使用 `CallWithProvider(callback interface{}, provider func() []*Entity) ([]interface{}, error)` 配合 `Provider(initializes ...interface{}) (func() []*Entity, error)` 方法实现该功能。

	ctxFunc := func() Context { return ctx }
	requestFunc := func() Request { return ctx.request }
	
	provider, _ := cc.Provider(ctxFunc, requestFunc)
	results, err := cc.CallWithProvider(func(userRepo repo.UserRepo, req Request) ([]repo.User, error) {
		// 这里我们注入的 Request 对象，只对当前 callback 有效
		userId := req.Input("user_id")
		users, err := userRepo.GetUser(userId)
		
		return users, err
	}, provider)

### AutoWire 结构体属性注入

使用 `AutoWire` 方法可以为结构体的属性注入其绑定的对象，要使用该特性，我们需要在需要依赖注入的结构体对象上添加 `autowire` 标签。

	type UserManager struct {
		UserRepo *UserRepo `autowire:"@" json:"-"`
		field1   string    `autowire:"version"`
		Field2   string    `json:"field2"`
	}

	manager := UserManager{}
	// 对 manager 执行 AutoWire 之后，会自动注入 UserRepo 和 field1 的值
	if err := c.AutoWire(&manager); err != nil {
		t.Error("test failed")
	}

结构体属性注入支持公开和私有字段的注入。如果对象是通过类型来注入的，使用 `autowire:"@"` 来标记属性；如果使用的是 `BindValue` 绑定的字符串为key的对象，则使用 `autowire:"Key名称"` 来标记属性。

> 由于 `AutoWire` 要修改对象，因此必须使用对象的指针，结构体类型必须使用 `&` 。

## 其它方法

### HasBound/HasBoundValue

方法签名 

	HasBound(key interface{}) bool
	HasBoundValue(key string) bool

用于判断指定的 Key 是否已经绑定过了。

### Keys

方法签名 

	Keys() []interface{}

获取所有绑定到 **Container** 中的对象信息。

### CanOverride

方法签名

	CanOverride(key interface{}) (bool, error)

判断指定的 Key 是否可以覆盖，重新绑定创建函数。

### Extend

`Extend` 并不是 **Container** 实例上的一个方法，而是一个独立的函数，用于从已有的 Container 生成一个新的 Container，新的 Container 继承已有 Container 所有的对象绑定。

	Extend(c Container) Container

容器继承之后，在依赖注入对象查找时，会优先从当前 Container 中查找，当找不到对象时，再从父对象查找。

> 在 Container 实例上个，有一个名为 `ExtendFrom(parent Container)` 的方法，该方法用于指定当前 Container 从 parent 继承。

## 示例项目

以下项目中使用了 `Container` 作为依赖注入管理库，感兴趣的可以参考一下。

- [Glacier](https://github.com/mylxsw/glacier) 一个应用管理框架，目前还没有写使用文档，该框架集成了 **Container**，用来管理框架的对象实例化。
- [Adanos-Alert](https://github.com/mylxsw/adanos-alert) 使用 Glacier 开发的一款报警系统，它侧重点并不是监控，而是报警，可以对各种报警信息进行聚合，按照配置规则来实现多样化的报警，一般用于配合 `Logstash` 来完成业务和错误日志的报警，配合`Prometheus`，`OpenFalcon` 等主流监控框架完成服务级的报警。目前还在开发中，但基本功能已经可用。
- [Sync](https://github.com/mylxsw/sync) 使用 Glacier 开发一款跨主机文件同步工具，拥有友好的 web 配置界面，使用 GRPC 实现不同服务器之间文件的同步。
