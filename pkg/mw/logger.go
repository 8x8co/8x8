package mw

import (
	"net/http"
	"time"

	"github.com/gernest/8x8/pkg/xl"
	"go.uber.org/zap"
)

func Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		end := time.Now()
		xl.Info(r.URL.Path,
			zap.String("method", r.Method),
			zap.String("host", r.Host),
			zap.Duration("duration", end.Sub(start)),
		)
	})
}
