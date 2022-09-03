package render

import (
	"encoding/xml"
	"net/http"
)

type XML struct {
	Data any
}

func (s *XML) Render(w http.ResponseWriter) error {
	s.WriteContentType(w)
	err := xml.NewEncoder(w).Encode(s.Data)
	return err
}

func (s *XML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "application/xml; charset=utf-8")
}
