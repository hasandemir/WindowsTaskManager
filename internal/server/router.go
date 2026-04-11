package server

import (
	"context"
	"net/http"
	"strings"
)

// paramKey is the context key for path parameters.
type paramKey struct{}

// Params is the map of named path parameters extracted from a route.
type Params map[string]string

// Param fetches a path parameter from a request, or "" if missing.
func Param(r *http.Request, name string) string {
	p, _ := r.Context().Value(paramKey{}).(Params)
	if p == nil {
		return ""
	}
	return p[name]
}

type route struct {
	method  string
	parts   []string // each part is literal or ":name"
	handler http.HandlerFunc
}

// Router is a tiny method+path router that supports `:param` segments.
type Router struct {
	routes     []route
	notFound   http.HandlerFunc
	middleware []func(http.HandlerFunc) http.HandlerFunc
}

func NewRouter() *Router {
	return &Router{
		notFound: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		},
	}
}

// Use appends a middleware that wraps every handler.
func (rt *Router) Use(mw func(http.HandlerFunc) http.HandlerFunc) {
	rt.middleware = append(rt.middleware, mw)
}

// SetNotFound overrides the default 404 handler.
func (rt *Router) SetNotFound(h http.HandlerFunc) { rt.notFound = h }

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// Handle registers a handler for a method+pattern.
func (rt *Router) Handle(method, pattern string, h http.HandlerFunc) {
	rt.routes = append(rt.routes, route{
		method:  strings.ToUpper(method),
		parts:   splitPath(pattern),
		handler: h,
	})
}

func (rt *Router) GET(p string, h http.HandlerFunc)    { rt.Handle("GET", p, h) }
func (rt *Router) POST(p string, h http.HandlerFunc)   { rt.Handle("POST", p, h) }
func (rt *Router) PUT(p string, h http.HandlerFunc)    { rt.Handle("PUT", p, h) }
func (rt *Router) DELETE(p string, h http.HandlerFunc) { rt.Handle("DELETE", p, h) }

func (rt *Router) match(method string, parts []string) (http.HandlerFunc, Params, bool) {
	for _, rte := range rt.routes {
		if rte.method != method {
			continue
		}
		if len(rte.parts) != len(parts) {
			continue
		}
		params := Params{}
		ok := true
		for i, part := range rte.parts {
			if strings.HasPrefix(part, ":") {
				params[part[1:]] = parts[i]
				continue
			}
			if part != parts[i] {
				ok = false
				break
			}
		}
		if ok {
			return rte.handler, params, true
		}
	}
	return nil, nil, false
}

// ServeHTTP implements http.Handler.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	h, params, ok := rt.match(r.Method, parts)
	if !ok {
		rt.notFound(w, r)
		return
	}
	if len(params) > 0 {
		ctx := context.WithValue(r.Context(), paramKey{}, params)
		r = r.WithContext(ctx)
	}
	for i := len(rt.middleware) - 1; i >= 0; i-- {
		h = rt.middleware[i](h)
	}
	h(w, r)
}
