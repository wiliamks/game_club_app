package service

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"gamer-club/backend/internal/models"
	"gamer-club/backend/internal/repository"
)

// UserService defines the interface for user operations and authentication
type UserService interface {
	Authenticate(username, password string) (string, *models.User, error)
	CreateUser(dto *models.CreateUserDTO) (*models.User, error)
	UpdateUser(id int, dto *models.UserUpdateDTO) error
	DeleteUser(id int) error
	GetAllUsers() ([]*models.User, error)
	GetUserByID(id int) (*models.User, error)
	ValidateToken(tokenStr string) (*models.User, error)
	GenerateTokenForUser(u *models.User) (string, error)
}

type userService struct {
	repo      repository.UserRepository
	jwtSecret []byte
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// NewUserService creates a new UserService implementation
func NewUserService(repo repository.UserRepository) UserService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_fallback_jwt_secret_please_change"
	}
	return &userService{
		repo:      repo,
		jwtSecret: []byte(secret),
	}
}

func (s *userService) Authenticate(username, password string) (string, *models.User, error) {
	u, err := s.repo.GetByUsername(username)
	if err != nil {
		return "", nil, err
	}
	if u == nil {
		return "", nil, errors.New("invalid username or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return "", nil, errors.New("invalid username or password")
	}

	// Create JWT token (expires in 24 hours)
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, u, nil
}

func (s *userService) CreateUser(dto *models.CreateUserDTO) (*models.User, error) {
	if dto.Username == "" || dto.Password == "" {
		return nil, errors.New("username and password cannot be empty")
	}
	if dto.Role != "admin" && dto.Role != "user" {
		return nil, errors.New("invalid role, must be 'admin' or 'user'")
	}

	existing, err := s.repo.GetByUsername(dto.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("username already exists")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := &models.User{
		Username:  dto.Username,
		Password:  string(hashed),
		Role:      dto.Role,
		AvatarURL: dto.AvatarURL,
	}

	if err := s.repo.Create(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *userService) UpdateUser(id int, dto *models.UserUpdateDTO) error {
	u, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("user not found")
	}

	if dto.Username != "" && dto.Username != u.Username {
		existing, err := s.repo.GetByUsername(dto.Username)
		if err != nil {
			return err
		}
		if existing != nil {
			return errors.New("username already exists")
		}
		u.Username = dto.Username
	}

	if dto.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashed)
	}

	u.AvatarURL = dto.AvatarURL

	return s.repo.Update(u)
}

func (s *userService) DeleteUser(id int) error {
	u, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("user not found")
	}
	if u.Role == "admin" {
		// Prevent deleting the last admin or standard admin protection
		// In a production-grade app, let's at least enforce that we can't delete ID 1
		if u.ID == 1 {
			return errors.New("cannot delete primary administrator account")
		}
	}
	return s.repo.Delete(id)
}

func (s *userService) GetAllUsers() ([]*models.User, error) {
	return s.repo.GetAll()
}

func (s *userService) GetUserByID(id int) (*models.User, error) {
	return s.repo.GetByID(id)
}

func (s *userService) ValidateToken(tokenStr string) (*models.User, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Load full details from repo to verify user still exists
	user, err := s.repo.GetByID(claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user no longer exists")
	}

	return user, nil
}

func (s *userService) GenerateTokenForUser(u *models.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
