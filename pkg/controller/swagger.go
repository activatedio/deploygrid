
package controller

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func NewSwaggerUIHandler(url *url.URL, prefix string) http.HandlerFunc {

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			r.SetURL(url)
			r.Out.URL.Path, _ = strings.CutPrefix(r.Out.URL.Path, prefix)
		},
	}

	return proxy.ServeHTTP

}
