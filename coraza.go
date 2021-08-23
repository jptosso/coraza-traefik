package coraza

import (
	"context"
	"io"
	"net/http"
	"strings"

	coraza "github.com/jptosso/coraza-waf"
)

// Config the plugin configuration.
type Config struct {
	Include    string `yaml:"include"`
	Directives string `yaml:"include"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// Coraza a plugin.
type Coraza struct {
	next http.Handler
	waf  *coraza.Waf
	// ...
}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &Coraza{
		next: next,
		waf:  coraza.NewWaf(),
	}, nil
}

func (e *Coraza) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	tx := e.waf.NewTransaction()
	it, err := tx.ProcessRequest(req)
	//TODO TEMPORARY FIX FOR A CORAZA BUG
	if req.Body == nil {
		req.Body = io.NopCloser(strings.NewReader(""))
	}
	if it != nil || err != nil {
		errorPage(rw, tx)
		return
	}
	rw = NewHTTPResponseInterceptor(rw, tx)

	e.next.ServeHTTP(rw, req)
}

func errorPage(rw http.ResponseWriter, tx *coraza.Transaction) {
	//TODO write som error page
	rw.WriteHeader(403)
	rw.Write([]byte("Access Denied!"))
}
