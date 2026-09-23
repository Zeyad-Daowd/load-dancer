package orderservice

import (
	"context"
	"fmt"

	db "github.com/zeyad-daowd/load-dancer/internal/orderservice/db/generated"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserRequest struct {
	Username string
	Password string
	Role     string
}
type User struct {
	ID       int32
	Username string
	Role     string
}

func HashPassword(password string) (string, error) {
	// Convert password string to byte slice
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

var ErrInvalidCreateUserRequest = fmt.Errorf("invalid create user request")

func (s *OrderService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	if req.Role != "admin" && req.Role != "user" {
		return nil, ErrInvalidCreateUserRequest
	}
	if req.Username == "" || req.Password == "" {
		return nil, ErrInvalidCreateUserRequest
	}
	hashed, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Username:     req.Username,
		PasswordHash: hashed,
		Role:         req.Role,
	})

	if err != nil {
		return nil, err
	}

	return &User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

var ErrNotFound = fmt.Errorf("user not found")

func (s *OrderService) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrNotFound
	}

	return &User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}
