package server

import (
	"net/http"
)

// Get - implementation of http get
func (rtr Router) Get(route string, handler http.HandlerFunc) {
	rtr.Engine.Get(route, handler)
}

// Patch - implementation of http patch
func (rtr Router) Patch(route string, handler http.HandlerFunc) {
	rtr.Engine.Patch(route, handler)
}

// Put - implementation of http put
func (rtr Router) Put(route string, handler http.HandlerFunc) {
	rtr.Engine.Put(route, handler)
}

// Post - implementation of http post
func (rtr Router) Post(route string, handler http.HandlerFunc) {
	rtr.Engine.Post(route, handler)
}

// Delete - implementation of http delete
func (rtr Router) Delete(route string, handler http.HandlerFunc) {
	rtr.Engine.Delete(route, handler)
}
