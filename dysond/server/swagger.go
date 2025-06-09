package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	docs "dysonprotocol.com/client/docs"
	"dysonprotocol.com/dysond/server/dwapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server/config"
)

// RegisterDysonServer provides a common function which registers APIs with API Server
// This includes both Swagger API (if enabled) and the dwapp handler for DysonScript web applications
func RegisterDysonServer(clientCtx client.Context, rtr *mux.Router, config config.APIConfig, scriptPattern string) error {

	// Register the DysonScript app handler
	// Use provided pattern or default if empty
	patternString := scriptPattern
	if patternString == "" {
		patternString = dwapp.DefaultDwAppPattern
	}

	// Register Swagger UI if enabled
	if config.Swagger {
		root, err := fs.Sub(docs.SwaggerUI, "swagger-ui")
		if err != nil {
			return err
		}

		staticServer := http.FileServer(http.FS(root))

		rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
		rtr.PathPrefix("/favicon.ico").Handler(staticServer)

	}

	if config.Enable {
		// Middleware to check path condition explicitly
		rtr.Use(func(next http.Handler) http.Handler {
			dwappHandler := dwapp.NewDefaultHandler(clientCtx, patternString)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !(strings.HasPrefix(r.URL.Path, "/dysonprotocol/") ||
					strings.HasPrefix(r.URL.Path, "/cosmos/") ||
					strings.HasPrefix(r.URL.Path, "/ibc/") ||
					strings.HasPrefix(r.URL.Path, "/favicon.ico") ||
					strings.HasPrefix(r.URL.Path, "/swagger/")) {
					// Condition matched: use alternative handler
					dwappHandler.ServeHTTP(w, r)
					return
				}
				// Else, continue with default handler
				next.ServeHTTP(w, r)
			})
		})
	}

	return nil
}
