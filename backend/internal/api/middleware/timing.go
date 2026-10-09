package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ResponseTimeHeader ajoute l'en-tête X-Response-Time (en millisecondes) : temps passé dans le
// serveur, hors réseau. Il permet aux tests de charge de distinguer la latence réseau du temps
// de traitement du backend.
const ResponseTimeHeader = "X-Response-Time"

type timingWriter struct {
	gin.ResponseWriter
	start time.Time
}

func (w *timingWriter) setHeader() {
	if !w.Written() {
		elapsed := float64(time.Since(w.start).Microseconds()) / 1000
		w.Header().Set(ResponseTimeHeader, strconv.FormatFloat(elapsed, 'f', 2, 64)+"ms")
	}
}

func (w *timingWriter) WriteHeaderNow() {
	w.setHeader()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *timingWriter) Write(data []byte) (int, error) {
	w.setHeader()
	return w.ResponseWriter.Write(data)
}

func (w *timingWriter) WriteString(s string) (int, error) {
	w.setHeader()
	return w.ResponseWriter.WriteString(s)
}

// ResponseTimeMiddleware mesure le temps de traitement de chaque requête.
func ResponseTimeMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer = &timingWriter{ResponseWriter: ctx.Writer, start: time.Now()}
		ctx.Next()
	}
}
