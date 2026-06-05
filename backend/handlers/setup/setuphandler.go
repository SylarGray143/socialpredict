package setup

import (
	"encoding/json"
	"net/http"

	"socialpredict/handlers"
	configsvc "socialpredict/internal/service/config"
)

func GetSetupHandler(configService configsvc.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if configService == nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(configService.Economics())
		if err != nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
		}
	}
}

type frontendChartsResponse struct {
	SigFigs int `json:"sigFigs"`
}

type frontendGameResponse struct {
	Mode string `json:"mode"`
}

type frontendConfigResponse struct {
	Charts         frontendChartsResponse         `json:"charts"`
	Game           frontendGameResponse           `json:"game"`
	OAuthProviders frontendOAuthProvidersResponse `json:"oauthProviders"`
}

type frontendOAuthProvidersResponse struct {
	Google bool `json:"google"`
}

func GetFrontendSetupHandler(configService configsvc.Service, oauthGoogleEnabled bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if configService == nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}

		response := frontendConfigResponse{
			Charts: frontendChartsResponse{
				SigFigs: configService.ChartSigFigs(),
			},
			Game: frontendGameResponse{
				Mode: configService.Game().Mode,
			},
			OAuthProviders: frontendOAuthProvidersResponse{
				Google: oauthGoogleEnabled,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			_ = handlers.WriteFailure(w, http.StatusInternalServerError, handlers.ReasonInternalError)
			return
		}
	}
}
