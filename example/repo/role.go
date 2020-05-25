package repo

import "database/sql"

type Role struct {
	ID   int
	Name string
}

type RoleRepo interface {
	GetUserRole(id int) (*Role, error)
}

type roleRepo struct {
	db *sql.DB
}

func NewRoleRepo(db *sql.DB) RoleRepo {
	return &roleRepo{db: db}
}

func (repo roleRepo) GetUserRole(id int) (*Role, error) {
	row := repo.db.QueryRow("SELECT id, name FROM role WHERE id=?", id)
	role := Role{}
	if err := row.Scan(&role.ID, &role.Name); err != nil {
		return nil, err
	}

	return &role, nil
}
