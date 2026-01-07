package user

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"deployment-platform/internal/models"
	"deployment-platform/internal/utils"
)

type Service interface {
	Register(ctx context.Context, email, password, name string) (*models.User, string, error)
	Login(ctx context.Context, email, password string) (*models.User, string, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) Register(ctx context.Context, email, password, name string) (*models.User, string, error) {
	// Check if user exists
	var existingUser models.User
	if err := s.db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return nil, "", errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user := &models.User{
		Email:    email,
		Password: string(hashedPassword),
		Name:     name,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, "", err
	}

	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *service) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}