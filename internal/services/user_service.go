package services

import (
	"errors"
	entity "narapulse-be/internal/models/entity"
	"narapulse-be/internal/repositories"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(req *entity.UserCreateRequest) (*entity.User, error)
	GetUserByID(id uint) (*entity.User, error)
	GetUserByEmail(email string) (*entity.User, error)
	UpdateUser(id uint, req *entity.UserUpdateRequest) (*entity.User, error)
	DeleteUser(id uint) error
	AuthenticateUser(email, password string) (*entity.User, error)
	GetAllUsers() ([]*entity.User, error)
}

type userService struct {
	userRepo repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) CreateUser(req *entity.UserCreateRequest) (*entity.User, error) {
	// Check if user already exists
	existingUser, _ := s.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &entity.User{
		Email:     req.Email,
		Username:  req.Username,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      "user",
		IsActive:  true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) GetUserByID(id uint) (*entity.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *userService) GetUserByEmail(email string) (*entity.User, error) {
	return s.userRepo.GetByEmail(email)
}

func (s *userService) UpdateUser(id uint, req *entity.UserUpdateRequest) (*entity.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) DeleteUser(id uint) error {
	_, err := s.userRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	return s.userRepo.Delete(id)
}

func (s *userService) GetAllUsers() ([]*entity.User, error) {
	return s.userRepo.GetAll()
}

func (s *userService) AuthenticateUser(email, password string) (*entity.User, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}