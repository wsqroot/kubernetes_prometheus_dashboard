package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"login/internal/model"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	db            *sql.DB
	userLookupSQL string
}

func NewUserRepository(db *sql.DB, userLookupSQL string) *UserRepository {
	return &UserRepository{
		db:            db,
		userLookupSQL: userLookupSQL,
	}
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.UserRecord, error) {
	row := r.db.QueryRowContext(ctx, r.userLookupSQL, username)

	var user model.UserRecord
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordText); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}

	return &user, nil
}
