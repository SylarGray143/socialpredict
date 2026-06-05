package authhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func googleOAuthConfig() *oauth2.Config {
	clientID := strings.TrimSpace(os.Getenv("OAUTH_GOOGLE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"))
	
	if clientID == "" || clientSecret == "" {
		return nil
	}

	baseURL := strings.TrimSpace(os.Getenv("OAUTH_CALLBACK_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("%s/v0/auth/callback/google", baseURL),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func fetchGoogleUserInfo(ctx context.Context, conf *oauth2.Config, token *oauth2.Token) (*googleUserInfo, error) {
	client := conf.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch user info, status: %d", resp.StatusCode)
	}

	var userInfo googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed decoding user info: %w", err)
	}

	if userInfo.ID == "" || userInfo.Email == "" {
		return nil, fmt.Errorf("missing required user info from Google")
	}

	return &userInfo, nil
}
