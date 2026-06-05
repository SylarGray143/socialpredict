package authhttp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"socialpredict/handlers"
	dusers "socialpredict/internal/domain/users"
	configsvc "socialpredict/internal/service/config"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

type oauthUserResolver interface {
	FindOrCreateOAuthUser(ctx context.Context, req dusers.OAuthUserRequest, initialAccountBalance int64) (*dusers.OAuthUserResult, error)
}

const oauthStateCookieName = "oauth_state"

// OAuthLoginHandler returns an HTTP handler that redirects to the OAuth provider's authorization URL.
func OAuthLoginHandler(configService configsvc.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := strings.ToLower(mux.Vars(r)["provider"])
		if provider != "google" {
			_ = handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonInvalidRequest)
			return
		}

		oauthConfig := googleOAuthConfig()
		if oauthConfig == nil {
			_ = handlers.WriteFailure(w, http.StatusNotFound, handlers.ReasonNotFound)
			return
		}

		state, err := generateOAuthState()
		if err != nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     oauthStateCookieName,
			Value:    state,
			Path:     "/",
			MaxAge:   300, // 5 minutes
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
		})

		url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	})
}

// OAuthCallbackHandler returns an HTTP handler that processes the OAuth provider's callback.
func OAuthCallbackHandler(resolver oauthUserResolver, jwtSigningKey []byte) http.Handler {
	key := append([]byte(nil), jwtSigningKey...)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := strings.ToLower(mux.Vars(r)["provider"])
		if provider != "google" {
			_ = handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonInvalidRequest)
			return
		}

		// Validate CSRF state.
		stateCookie, err := r.Cookie(oauthStateCookieName)
		if err != nil || stateCookie.Value == "" {
			_ = handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonInvalidRequest)
			return
		}
		if r.URL.Query().Get("state") != stateCookie.Value {
			_ = handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonInvalidRequest)
			return
		}

		// Clear the state cookie.
		http.SetCookie(w, &http.Cookie{
			Name:     oauthStateCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		// Check for error response from provider.
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			_ = handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonAuthorizationDenied)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			_ = handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonInvalidRequest)
			return
		}

		oauthConfig := googleOAuthConfig()
		if oauthConfig == nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		// Exchange code for token.
		token, err := oauthConfig.Exchange(r.Context(), code)
		if err != nil {
			_ = handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonAuthorizationDenied)
			return
		}

		// Fetch user info from provider.
		userInfo, err := fetchGoogleUserInfo(r.Context(), oauthConfig, token)
		if err != nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		if resolver == nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		// Read initial balance from config if available.
		var initialBalance int64
		if balance, ok := r.Context().Value(ctxKeyInitialBalance).(int64); ok {
			initialBalance = balance
		}

		result, err := resolver.FindOrCreateOAuthUser(r.Context(), dusers.OAuthUserRequest{
			Provider:   provider,
			ProviderID: userInfo.ID,
			Email:      userInfo.Email,
			Name:       userInfo.Name,
		}, initialBalance)
		if err != nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		if len(key) == 0 {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		jwtToken, err := generateOAuthJWT(result.Username, key)
		if err != nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		// Redirect to frontend with auth data.
		frontendURL := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
		if frontendURL == "" {
			frontendURL = "http://localhost:5173"
		}
		redirectURL := fmt.Sprintf("%s/auth/callback?token=%s&username=%s&usertype=%s&mustChangePassword=%t",
			frontendURL, jwtToken, result.Username, result.UserType, result.MustChangePassword)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
	})
}

// OAuthCallbackHandlerWithConfig wraps OAuthCallbackHandler to inject the initial balance from config.
func OAuthCallbackHandlerWithConfig(resolver oauthUserResolver, configService configsvc.Service, jwtSigningKey []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var balance int64
		if configService != nil {
			balance = configService.Economics().User.InitialAccountBalance
		}
		ctx := context.WithValue(r.Context(), ctxKeyInitialBalance, balance)
		OAuthCallbackHandler(resolver, jwtSigningKey).ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func generateOAuthJWT(username string, jwtKey []byte) (string, error) {
	if len(jwtKey) == 0 {
		return "", fmt.Errorf("missing JWT signing key")
	}

	type oauthClaims struct {
		Username string `json:"username"`
		jwt.StandardClaims
	}

	claims := &oauthClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}
