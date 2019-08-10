# container

[![Build Status](https://www.travis-ci.org/mylxsw/container.svg?branch=master)](https://www.travis-ci.org/mylxsw/container)
[![Coverage Status](https://coveralls.io/repos/github/mylxsw/container/badge.svg?branch=master)](https://coveralls.io/github/mylxsw/container?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/mylxsw/container)](https://goreportcard.com/report/github.com/mylxsw/container)
[![codecov](https://codecov.io/gh/mylxsw/container/branch/master/graph/badge.svg)](https://codecov.io/gh/mylxsw/container)
[![Sourcegraph](https://sourcegraph.com/github.com/mylxsw/container/-/badge.svg)](https://sourcegraph.com/github.com/mylxsw/container?badge)
[![GitHub](https://img.shields.io/github/license/mylxsw/container.svg)](https://github.com/mylxsw/container)


Container is a Go dependency injection library.

	c := container.New()

	c.BindValue("conn_str", "root:root@/my_db?charset=utf8")
	c.Singleton(func(c *container.Container) (*UserRepo, error) {
		connStr, err := c.Get("conn_str")
		if err != nil {
			return nil, err
		}

		return &UserRepo{connStr: connStr.(string)}, nil
	})
	c.Prototype(func(userRepo *UserRepo) (*UserService, error) {
		return &UserService{repo: userRepo}, nil
	})

	if err := c.Resolve(func(userService *UserService) {
		if userService.GetUser() != expectedValue {
			t.Error("test failed")
		}
	}); err != nil {
		panic(err)
	}

	userService, err := c.Get((*UserService)(nil))
	if err != nil {
		panic(err)
	}

	if userService.(*UserService).GetUser() != expectedValue {
		panic(err)
	}