package coraza

import (
	"fmt"
	"net/http"

	coraza "github.com/jptosso/coraza-waf"
)

type HTTPResponseInterceptor struct {
	http.ResponseWriter
	StatusCode int
	tx         *coraza.Transaction
}

func NewHTTPResponseInterceptor(w http.ResponseWriter, tx *coraza.Transaction) *HTTPResponseInterceptor {
	return &HTTPResponseInterceptor{w, http.StatusOK, tx}
}

func (i *HTTPResponseInterceptor) WriteHeader(code int) {

	if i.tx.Interruption.Action == "deny" {
		i.StatusCode = i.tx.Interruption.Status
		i.ResponseWriter.WriteHeader(i.tx.Interruption.Status)
	} else {
		i.StatusCode = code
		i.ResponseWriter.WriteHeader(code)
	}

}

func (i *HTTPResponseInterceptor) Write(data []byte) (int, error) {
	c, err := i.tx.ResponseBodyBuffer.Write(data)
	if err != nil {
		return c, err
	}
	if i.tx.Interrupted() {
		return 0, fmt.Errorf("transaction interrupted")
	}
	return i.ResponseWriter.Write(data)
}

func (i *HTTPResponseInterceptor) Header() http.Header {
	for k, vs := range i.ResponseWriter.Header() {
		for _, v := range vs {
			i.tx.AddResponseHeader(k, v)
		}
	}
	if it := i.tx.ProcessResponseHeaders(i.StatusCode, "http/1.1"); it != nil {
		h := http.Header{}
		h.Add("Content-Type", "text/html")
		return h
	} else {
		return i.ResponseWriter.Header()
	}
}
