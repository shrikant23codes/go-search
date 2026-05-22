package middleware

import "net/http"

func Auth(apiKeys []string) func(http.Handler) http.Handler {
	// we use struct{} as it takes 0 bytes of mem, but bool takes 1 byte
	// Called once at startup
	keySet := make(map[string]struct{}, len(apiKeys))
	for _, key := range apiKeys {
		keySet[key] = struct{}{} // empty struct
	}

	// Called for every request
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
			}

			key := r.Header.Get("X-API-Key")
			if _, ok := keySet[key]; !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
