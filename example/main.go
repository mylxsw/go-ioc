package main

import (
	"database/sql"
	"github.com/mylxsw/go-ioc"
	"log"

	"github.com/mylxsw/container/example/repo"
	_ "github.com/proullon/ramsql/driver"
)

type Demo struct {
	UserRepo repo.UserRepo `autowire:"@"`
	roleRepo repo.RoleRepo `autowire:"@"` // 支持 private 字段
}

func main() {

	cc := ioc.New()

	// 绑定对象创建函数
	cc.MS(repo.NewUserRepo)
	cc.MustSingleton(repo.NewRoleRepo)
	cc.MS(func() (*sql.DB, error) {
		db, err := sql.Open("ramsql", "database address")
		if err != nil {
			return nil, err
		}

		// add some test data
		_, _ = db.Exec("CREATE TABLE user(id int primary key, name varchar, role_id int)")
		_, _ = db.Exec("INSERT INTO user(id, name, role_id) VALUES(1, 'mylxsw', 1)")

		_, _ = db.Exec("CREATE TABLE role(id int primary key, name varchar)")
		_, _ = db.Exec("INSERT INTO role(id, name) VALUES(1, 'manager')")

		return db, nil
	})

	// 使用 ResolveWithError
	err := cc.R(func(userRepo repo.UserRepo, roleRepo repo.RoleRepo) error {
		user, err := userRepo.GetUser(1)
		if err != nil {
			return err
		}

		role, _ := roleRepo.GetUserRole(user.RoleID)
		log.Printf("resole: id=%d, name=%s, role=%s", user.ID, user.Name, role.Name)

		return nil
	})
	if err != nil {
		panic(err)
	}

	// 使用 AutoWire 初始化结构体
	{
		demo := Demo{}
		//cc.Must(cc.AutoWire(&demo))
		cc.MW(&demo)

		user, err := demo.UserRepo.GetUser(1)
		if err != nil {
			panic(err)
		}

		role, _ := demo.roleRepo.GetUserRole(user.RoleID)
		log.Printf("autowire: id=%d, name=%s, role=%s", user.ID, user.Name, role.Name)
	}
}
