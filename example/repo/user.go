package repo

import (
	"database/sql"

	"github.com/mylxsw/container"
)

type User struct {
	ID     int
	Name   string
	RoleID int
}

type UserRepo interface {
	GetUser(id int) (*User, error)
}

type userRepo struct {
	cc container.Container
	db *sql.DB
}

func (repo userRepo) GetUser(id int) (*User, error) {
	row := repo.db.QueryRow("SELECT id, name, role_id FROM user WHERE id=?", id)
	user := User{}
	if err := row.Scan(&user.ID, &user.Name, &user.RoleID); err != nil {
		return nil, err
	}

	return &user, nil
}

func NewUserRepo(cc container.Container, db *sql.DB) UserRepo {
	return &userRepo{cc: cc, db: db}
}
