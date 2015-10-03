// Package renderable makes APIs responses more convenient.
//
// Responses can be rendered as multiple content types.  HTML is the
// default, but any supported content type can be selected via content
// negotiation.
//
//  http.handleFunc("/names", HandleRequest(ListNames))
//
//  func ListNames(w http.ResponseWriter, req *http.Request) (interface{}, error) {
//    return Renderable{
//      Data: []string{"Jane", "Joe", "Ann", "Pip"},
//    }, nil
//  }
package renderable

import (
	"encoding/json"
	"fmt"
	"github.com/golang/gddo/httputil"
	"html/template"
	"log"
	"net/http"
)

type Renderable struct {
	Status   int
	Metadata map[string]interface{}
	Data     interface{}
	Template *template.Template
}

func RenderableStatus(status int) Renderable {
	return Renderable{
		Status: status,
		Data: httpStatus{
			Status:  status,
			Message: http.StatusText(status),
		},
	}
}

type httpStatus struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func HandleRequest(handler func(http.ResponseWriter, *http.Request) (interface{}, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		contentType := httputil.NegotiateContentType(req, []string{"text/html", "application/json"}, "")

		data, err := handler(w, req)
		if err != nil {
			log.Printf("Error: %s: %s\n", req.URL.Path, err)
			render(w, req, contentType, RenderableStatus(http.StatusInternalServerError))
			return
		}

		renderable, ok := data.(Renderable)
		if !ok {
			panic("not implemented")
		}
		render(w, req, contentType, renderable)
	}
}

func render(w http.ResponseWriter, req *http.Request, contentType string, renderable Renderable) {
	if renderable.Status == 0 {
		renderable.Status = 200
	}
	w.WriteHeader(renderable.Status)

	switch contentType {
	case "text/html":
		if renderable.Template != nil {
			err := renderable.Template.Execute(w, renderable)
			if err != nil {
				// FIXME: we might be in a partial response here, i.e. we
				//        probably didn't return valid html.
				log.Printf("Error: rendering %s: %s\n", req.URL.Path, err)
			}
		} else {
			fmt.Fprint(w, http.StatusText(renderable.Status))
		}
	case "application/json":
		var err error

		pretty := req.URL.Query().Get("pretty") == "true"
		if pretty {
			data, err := json.MarshalIndent(renderable.Data, "", "  ")
			if err == nil {
				w.Write(data)
				w.Write([]byte{'\n'})
			}
		} else {
			encoder := json.NewEncoder(w)
			err = encoder.Encode(renderable.Data)
		}

		if err != nil {
			log.Printf("Error: rendering %s: %s\n", req.URL.Path, err)
		}
	default:
		status := http.StatusUnsupportedMediaType
		http.Error(w, http.StatusText(status), status)
	}
}
