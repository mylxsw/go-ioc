package ioc_test

import (
	"github.com/mylxsw/go-ioc"
	"strconv"
	"testing"
)

func buildContainer() ioc.Container {
	cc := ioc.New()
	cc.MS(func() *RoleService { return &RoleService{} })
	cc.MS(func(userRepo *UserRepo) *UserService { return &UserService{repo: userRepo} })
	cc.MS(func() *UserRepo { return &UserRepo{connStr: "oops, no connection"} })
	cc.MS(func() InterfaceDemo { return demo1{} })
	cc.MV("version", "1.0.0")
	for i := 0; i < 100; i++ {
		cc.MV("version-"+strconv.Itoa(i), "1.0.0")
	}
	cc.MS(func(userRepo *UserRepo, cc ioc.Container) *UserManager {
		m := &UserManager{UserRepo: userRepo}
		cc.MustAutoWire(m)
		m.Field2 = "Hello, world"
		return m
	})

	return cc
}

// 3161624	       366.9 ns/op
// 3309002	       356.6 ns/op
// 985084	      1213 ns/op
// 646950	      2054 ns/op
// 5368746	       219.3 ns/op
func BenchmarkContainerImpl_Resolve(b *testing.B) {
	cc := buildContainer()
	for i := 0; i < b.N; i++ {
		cc.MustResolve(func(userManager *UserManager) {
			// DO NOTHING
		})
	}
}

// 3883568	       281.9 ns/op
// 5913072	       190.9 ns/op
// 1651430	       687.5 ns/op
// 413727	      2599 ns/op
func BenchmarkContainerImpl_Keys(b *testing.B) {
	cc := buildContainer()
	for i := 0; i < b.N; i++ {
		cc.Keys()
	}
}
