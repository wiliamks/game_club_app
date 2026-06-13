package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"gamer-club/backend/internal/models"
)

// UserRepository defines the interface for user database operations
type UserRepository interface {
	Create(u *models.User) error
	GetByID(id int) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(u *models.User) error
	Delete(id int) error
	GetAll() ([]*models.User, error)
}

type userRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new SQLite implementation of UserRepository
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(u *models.User) error {
	query := "INSERT INTO users (username, password, role, avatar_url) VALUES (?, ?, ?, ?)"
	res, err := r.db.Exec(query, u.Username, u.Password, u.Role, u.AvatarURL)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	u.ID = int(id)
	return nil
}

func (r *userRepository) GetByID(id int) (*models.User, error) {
	query := "SELECT id, username, password, role, COALESCE(avatar_url, '') FROM users WHERE id = ?"
	row := r.db.QueryRow(query, id)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.AvatarURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil, nil when not found to easily handle it
		}
		return nil, fmt.Errorf("failed to scan user by ID: %w", err)
	}
	return &u, nil
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	query := "SELECT id, username, password, role, COALESCE(avatar_url, '') FROM users WHERE username = ?"
	row := r.db.QueryRow(query, username)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.AvatarURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan user by username: %w", err)
	}
	return &u, nil
}

func (r *userRepository) Update(u *models.User) error {
	query := "UPDATE users SET username = ?, password = ?, avatar_url = ? WHERE id = ?"
	_, err := r.db.Exec(query, u.Username, u.Password, u.AvatarURL, u.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *userRepository) Delete(id int) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (r *userRepository) GetAll() ([]*models.User, error) {
	query := "SELECT id, username, role, COALESCE(avatar_url, '') FROM users"
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("failed to scan user in list: %w", err)
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
