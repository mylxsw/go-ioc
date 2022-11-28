module github.com/mylxsw/container/example

go 1.19

require (
	github.com/mylxsw/go-ioc v0.0.0
	github.com/proullon/ramsql v0.0.0-20181213202341-817cee58a244
)

replace (
	github.com/mylxsw/go-ioc v0.0.0 => ../
)