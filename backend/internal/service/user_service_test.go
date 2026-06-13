package service

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"gamer-club/backend/internal/models"
)

// mockUserRepository implements repository.UserRepository for testing
type mockUserRepository struct {
	users  map[int]*models.User
	nextID int
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:  make(map[int]*models.User),
		nextID: 1,
	}
}

func (m *mockUserRepository) Create(u *models.User) error {
	u.ID = m.nextID
	m.nextID++
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepository) GetByID(id int) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepository) GetByUsername(username string) (*models.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) Update(u *models.User) error {
	if _, ok := m.users[u.ID]; !ok {
		return errors.New("user not found")
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepository) Delete(id int) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) GetAll() ([]*models.User, error) {
	var list []*models.User
	for _, u := range m.users {
		list = append(list, u)
	}
	return list, nil
}

func TestUserService(t *testing.T) {
	repo := newMockUserRepository()
	s := NewUserService(repo)

	// Create User Test
	dto := &models.CreateUserDTO{
		Username: "goku",
		Password: "kamehameha",
		Role:     "user",
	}

	user, err := s.CreateUser(dto)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Username != "goku" {
		t.Errorf("expected username to be goku, got %s", user.Username)
	}

	// Duplicate Create User Test
	_, err = s.CreateUser(dto)
	if err == nil {
		t.Error("expected duplicate username error, got nil")
	}

	// Authenticate Success Test
	token, authedUser, err := s.Authenticate("goku", "kamehameha")
	if err != nil {
		t.Fatalf("auth failed: %v", err)
	}
	if authedUser.ID != user.ID {
		t.Errorf("expected authed user ID to be %d, got %d", user.ID, authedUser.ID)
	}
	if token == "" {
		t.Error("expected token to be non-empty")
	}

	// Authenticate Fail Test
	_, _, err = s.Authenticate("goku", "wrong_password")
	if err == nil {
		t.Error("expected authentication failure, got nil")
	}

	// Validate Token Test
	validated, err := s.ValidateToken(token)
	if err != nil {
		t.Fatalf("token validation failed: %v", err)
	}
	if validated.ID != user.ID {
		t.Errorf("expected validated user ID %d, got %d", user.ID, validated.ID)
	}

	// Validation tests for CreateUser and UpdateUser
	_, err = s.CreateUser(&models.CreateUserDTO{Username: "", Password: "123"})
	if err == nil {
		t.Error("expected error with empty username")
	}
	_, err = s.CreateUser(&models.CreateUserDTO{Username: "test", Password: "123", Role: "invalid"})
	if err == nil {
		t.Error("expected error with invalid role")
	}

	err = s.UpdateUser(999, &models.UserUpdateDTO{Username: "new"})
	if err == nil {
		t.Error("expected error updating non-existent user")
	}

	// Update User Test
	updateDTO := &models.UserUpdateDTO{
		Username: "goku_super",
		Password: "spiritbomb",
	}
	err = s.UpdateUser(user.ID, updateDTO)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	updated, _ := repo.GetByID(user.ID)
	if updated.Username != "goku_super" {
		t.Errorf("expected updated username goku_super, got %s", updated.Username)
	}

	// Check password changed
	err = bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("spiritbomb"))
	if err != nil {
		t.Error("expected bcrypt password comparison to match")
	}

	// Delete Admin Guard Test (cannot delete primary administrator account with ID 1)
	adminUser := &models.User{
		ID:       1,
		Username: "admin",
		Role:     "admin",
	}
	repo.users[1] = adminUser

	err = s.DeleteUser(1)
	if err == nil {
		t.Error("expected error when deleting primary admin user, got nil")
	}

	// Delete Regular User Test
	userToDelete := &models.User{
		ID:       2,
		Username: "goku_delete",
		Role:     "user",
	}
	repo.users[2] = userToDelete
	err = s.DeleteUser(2)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// GetAllUsers Test
	allUsers, err := s.GetAllUsers()
	if err != nil {
		t.Fatalf("failed to get all users: %v", err)
	}
	if len(allUsers) == 0 {
		t.Error("expected users to be non-empty")
	}

	// GetUserByID Test
	foundUser, err := s.GetUserByID(1)
	if err != nil {
		t.Fatalf("failed to get user by ID: %v", err)
	}
	if foundUser == nil || foundUser.Username != "admin" {
		t.Error("expected to find admin user by ID 1")
	}

	// GenerateTokenForUser Test
	freshToken, err := s.GenerateTokenForUser(foundUser)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if freshToken == "" {
		t.Error("expected generated token to be non-empty")
	}
}
