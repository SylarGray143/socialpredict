package users

import (
	"context"
	"fmt"
	"net/mail"
)

// CreateSelfRegisteredUser creates a regular user from a self-registration request.
// Unlike admin-managed creation, the user supplies their own username, email, and password.
func (s *Service) CreateSelfRegisteredUser(ctx context.Context, req SelfRegistrationRequest, initialAccountBalance int64) (*SelfRegistrationResult, error) {
	if err := validateUsername(req.Username); err != nil {
		return nil, err
	}
	if err := validateEmail(req.Email); err != nil {
		return nil, err
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	uniqueness, err := s.userUniquenessRepository()
	if err != nil {
		return nil, err
	}

	if exists, err := uniqueness.UsernameExists(ctx, req.Username); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrUserAlreadyExists
	}

	if exists, err := uniqueness.EmailExists(ctx, req.Email); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrEmailAlreadyExists
	}

	var displayName string
	if req.DisplayName != "" {
		if exists, err := uniqueness.DisplayNameExists(ctx, req.DisplayName); err != nil {
			return nil, err
		} else if exists {
			return nil, ErrUserAlreadyExists
		}
		displayName = req.DisplayName
	} else {
		var err error
		displayName, err = uniqueDisplayName(ctx, uniqueness)
		if err != nil {
			return nil, err
		}
	}
	apiKey, err := uniqueAPIKey(ctx, uniqueness)
	if err != nil {
		return nil, err
	}

	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Username:              req.Username,
		DisplayName:           displayName,
		Email:                 req.Email,
		APIKey:                apiKey,
		PasswordHash:          passwordHash,
		UserType:              string(UserTypeRegular),
		ModeratorStatus:       ModeratorStatusNone,
		InitialAccountBalance: initialAccountBalance,
		AccountBalance:        initialAccountBalance,
		PersonalEmoji:         randomEmoji(),
		MustChangePassword:    false,
		AuthProvider:          "local",
	}

	if exists, err := uniqueness.AnyUserIdentityExists(ctx, user.Username, user.DisplayName, user.Email, user.APIKey); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrUserAlreadyExists
	}

	writer, err := s.userWriter()
	if err != nil {
		return nil, err
	}
	if err := writer.Create(ctx, user); err != nil {
		return nil, err
	}

	return &SelfRegistrationResult{
		Username: user.Username,
		UserType: user.UserType,
	}, nil
}

func validateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}
