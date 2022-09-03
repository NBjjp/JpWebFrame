package render

import (
	"github.com/NBjjp/JpWebFrame/internal/bytesconv"
	"html/template"
	"net/http"
)

type HTML struct {
	Data       any
	Name       string
	Template   *template.Template
	IsTemplate bool
}

type HTMLRender struct {
	Template *template.Template
}

func (h *HTML) Render(w http.ResponseWriter) error {
	h.WriteContentType(w)
	//_, err := w.Write([]byte(h.Data))
	//如果是模板
	if h.IsTemplate {
		err := h.Template.ExecuteTemplate(w, h.Name, h.Data)
		return err
	}
	_, err := w.Write(bytesconv.StringtoBytes(h.Data.(string)))
	return err
}

func (h *HTML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "text/html;charset=utf-8")
}
