package users

import (
	"context"
	"fmt"
)

// OAuthUserResult contains the user fields returned after OAuth login or creation.
type OAuthUserResult struct {
	Username           string
	UserType           string
	MustChangePassword bool
	IsNewUser          bool
}

// OAuthUserFinder exposes the lookup needed for OAuth user resolution.
type OAuthUserFinder interface {
	FindByOAuthProvider(ctx context.Context, provider, authID string) (*User, error)
}

// FindOrCreateOAuthUser resolves an OAuth identity to an existing user or creates a new one.
func (s *Service) FindOrCreateOAuthUser(ctx context.Context, req OAuthUserRequest, initialAccountBalance int64) (*OAuthUserResult, error) {
	if req.Provider == "" || req.ProviderID == "" {
		return nil, ErrInvalidUserData
	}
	if req.Email == "" {
		return nil, ErrInvalidEmail
	}

	oauthFinder, err := s.oauthUserFinder()
	if err != nil {
		return nil, err
	}

	// Check if a user already exists with this OAuth identity.
	existing, err := oauthFinder.FindByOAuthProvider(ctx, req.Provider, req.ProviderID)
	if err == nil && existing != nil {
		return &OAuthUserResult{
			Username:           existing.Username,
			UserType:           existing.UserType,
			MustChangePassword: existing.MustChangePassword,
			IsNewUser:          false,
		}, nil
	}

	// No existing OAuth user — create a new one.
	uniqueness, err := s.userUniquenessRepository()
	if err != nil {
		return nil, err
	}

	username, err := deriveOAuthUsername(ctx, uniqueness, req.Email)
	if err != nil {
		return nil, err
	}

	displayName, err := uniqueDisplayName(ctx, uniqueness)
	if err != nil {
		return nil, err
	}
	apiKey, err := uniqueAPIKey(ctx, uniqueness)
	if err != nil {
		return nil, err
	}

	// If the email is already registered (e.g. local user), generate a unique one.
	email := req.Email
	if exists, checkErr := uniqueness.EmailExists(ctx, email); checkErr != nil {
		return nil, checkErr
	} else if exists {
		email, err = uniqueEmail(ctx, uniqueness)
		if err != nil {
			return nil, err
		}
	}

	user := &User{
		Username:              username,
		DisplayName:           displayName,
		Email:                 email,
		APIKey:                apiKey,
		PasswordHash:          "", // OAuth users have no password
		UserType:              string(UserTypeRegular),
		ModeratorStatus:       ModeratorStatusNone,
		InitialAccountBalance: initialAccountBalance,
		AccountBalance:        initialAccountBalance,
		PersonalEmoji:         randomEmoji(),
		MustChangePassword:    false,
		AuthProvider:          req.Provider,
		AuthID:                req.ProviderID,
	}

	writer, err := s.userWriter()
	if err != nil {
		return nil, err
	}
	if err := writer.Create(ctx, user); err != nil {
		return nil, err
	}

	return &OAuthUserResult{
		Username:           user.Username,
		UserType:           user.UserType,
		MustChangePassword: false,
		IsNewUser:          true,
	}, nil
}

// deriveOAuthUsername creates a unique username from the email local part.
func deriveOAuthUsername(ctx context.Context, uniqueness UserUniquenessRepository, email string) (string, error) {
	// Extract the local part (before @) as the base username.
	base := emailLocalPart(email)
	if base == "" {
		base = "user"
	}

	// Lowercase and strip non-alphanumeric characters for safety.
	cleaned := cleanUsername(base)
	if len(cleaned) < 3 {
		cleaned = "user"
	}
	if len(cleaned) > 20 {
		cleaned = cleaned[:20]
	}

	// Try the base name first, then append numbers.
	candidate := cleaned
	for i := 0; i < 100; i++ {
		exists, err := uniqueness.UsernameExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s%d", cleaned, i+1)
	}

	return "", fmt.Errorf("could not generate unique username from email")
}

func emailLocalPart(email string) string {
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}

func cleanUsername(s string) string {
	var result []byte
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else if c >= 'A' && c <= 'Z' {
			result = append(result, c+32) // lowercase
		}
	}
	return string(result)
}

// oauthUserFinder returns the OAuth finder port, which must be implemented by the repository.
func (s *Service) oauthUserFinder() (OAuthUserFinder, error) {
	if s == nil || s.reader == nil {
		return nil, ErrInvalidUserData
	}
	finder, ok := s.reader.(OAuthUserFinder)
	if !ok {
		return nil, fmt.Errorf("repository does not support OAuth user lookup")
	}
	return finder, nil
}
