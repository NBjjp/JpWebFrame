package render

import "net/http"

type Render interface {
	Render(w http.ResponseWriter) error
	WriteContentType(w http.ResponseWriter)
}

func writeContentType(w http.ResponseWriter, values string) {
	w.Header().Set("Content-Type", values)
}
