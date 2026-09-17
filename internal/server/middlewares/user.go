package middlewares

import "net/http"

func (m Middleware) WithUser(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		ow := w

		userID := r.Header.Get("X-User-ID")

		if userID == "" {
			m.logger.Error("Header X-User-ID was not provided")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		m.logger.Debug("User set to ctx", "userid", userID)

		// Устанавливаем идентификатор пользователя в контекст
		ctx := m.service.SetUserIdToCtx(r.Context(), userID)
		r = r.WithContext(ctx)

		h.ServeHTTP(ow, r)
	}

	return http.HandlerFunc(logFn)
}
