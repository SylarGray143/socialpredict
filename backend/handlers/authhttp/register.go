package authhttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"socialpredict/handlers"
	dusers "socialpredict/internal/domain/users"
	configsvc "socialpredict/internal/service/config"
	"socialpredict/security"
)

type selfRegistrar interface {
	CreateSelfRegisteredUser(ctx context.Context, req dusers.SelfRegistrationRequest, initialAccountBalance int64) (*dusers.SelfRegistrationResult, error)
}

type registerRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=30,username"`
	DisplayName string `json:"displayname" validate:"omitempty,min=1,max=50"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=1"`
}

type registerResponse struct {
	Username string `json:"username"`
	UserType string `json:"usertype"`
}

// RegisterHandler returns an HTTP handler for POST /v0/register.
func RegisterHandler(registrar selfRegistrar, securityService *security.SecurityService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			_ = handlers.WriteFailure(w, http.StatusMethodNotAllowed, handlers.ReasonMethodNotAllowed)
			return
		}

		req, err := decodeRegisterRequest(r)
		if err != nil {
			_ = handlers.WriteFailure(w, http.StatusBadRequest, handlers.ReasonInvalidRequest)
			return
		}

		req, err = validateAndSanitizeRegister(securityService, req)
		if err != nil {
			_ = handlers.WriteFailureWithMessage(w, http.StatusBadRequest, handlers.ReasonValidationFailed, err.Error())
			return
		}

		if registrar == nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		// Read initial account balance from config service attached to context,
		// or fall back to zero. The handler accepts the registrar which is the
		// users service — config is read at the route-wiring layer.
		initialBalance := initialBalanceFromContext(r.Context())

		result, createErr := registrar.CreateSelfRegisteredUser(r.Context(), dusers.SelfRegistrationRequest{
			Username:    req.Username,
			DisplayName: req.DisplayName,
			Email:       req.Email,
			Password:    req.Password,
		}, initialBalance)
		if createErr != nil {
			writeRegistrationError(w, createErr)
			return
		}

		_ = handlers.WriteResult(w, http.StatusCreated, registerResponse{
			Username: result.Username,
			UserType: result.UserType,
		})
	})
}

// RegisterHandlerWithConfig returns an HTTP handler that resolves initial balance from a config service.
func RegisterHandlerWithConfig(registrar selfRegistrar, configService configsvc.Service, securityService *security.SecurityService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var balance int64
		if configService != nil {
			balance = configService.Economics().User.InitialAccountBalance
		}
		ctx := context.WithValue(r.Context(), ctxKeyInitialBalance, balance)
		RegisterHandler(registrar, securityService).ServeHTTP(w, r.WithContext(ctx))
	})
}

type contextKey string

const ctxKeyInitialBalance contextKey = "initialBalance"

func initialBalanceFromContext(ctx context.Context) int64 {
	if v, ok := ctx.Value(ctxKeyInitialBalance).(int64); ok {
		return v
	}
	return 0
}

func decodeRegisterRequest(r *http.Request) (registerRequest, error) {
	var req registerRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return registerRequest{}, fmt.Errorf("error reading request body")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return registerRequest{}, fmt.Errorf("error reading request body")
	}
	return req, nil
}

func validateAndSanitizeRegister(securityService *security.SecurityService, req registerRequest) (registerRequest, error) {
	if securityService == nil {
		return req, fmt.Errorf("security service unavailable")
	}

	sanitizedUsername, err := securityService.Sanitizer.SanitizeUsername(req.Username)
	if err != nil {
		return req, err
	}
	req.Username = sanitizedUsername

	if req.DisplayName != "" {
		sanitizedDisplayName, err := securityService.Sanitizer.SanitizeDisplayName(req.DisplayName)
		if err != nil {
			return req, err
		}
		req.DisplayName = sanitizedDisplayName
	}

	sanitizedPassword, err := securityService.Sanitizer.SanitizePassword(req.Password)
	if err != nil {
		return req, err
	}
	req.Password = sanitizedPassword

	if err := securityService.Validator.ValidateStruct(req); err != nil {
		return req, err
	}

	return req, nil
}

func writeRegistrationError(w http.ResponseWriter, err error) {
	if errors.Is(err, dusers.ErrUserAlreadyExists) {
		_ = handlers.WriteFailureWithMessage(w, http.StatusConflict, handlers.ReasonValidationFailed, "User already exists")
		return
	}
	if errors.Is(err, dusers.ErrEmailAlreadyExists) {
		_ = handlers.WriteFailureWithMessage(w, http.StatusConflict, handlers.ReasonValidationFailed, "Email already exists")
		return
	}
	if errors.Is(err, dusers.ErrInvalidEmail) {
		_ = handlers.WriteFailureWithMessage(w, http.StatusBadRequest, handlers.ReasonValidationFailed, "Invalid email address")
		return
	}
	if errors.Is(err, dusers.ErrInvalidUserData) {
		_ = handlers.WriteFailureWithMessage(w, http.StatusBadRequest, handlers.ReasonValidationFailed, "Invalid user data")
		return
	}

	_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
}
