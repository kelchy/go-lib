package client

import (
	"io"
	"bytes"
	"net/http"
	"encoding/json"
	"github.com/kelchy/go-lib/log"
)

// Res - response instance from http call
type Res struct {
	Response	http.Response
	Error		error
	log		log.Log
	HTML		string
	JSON		json.RawMessage
}

// HTMLparse - method to return the html content of response
func (r *Res) HTMLparse() {
	var data bytes.Buffer
	if r.Error != nil {
		return
	}
	defer r.Response.Body.Close()
	_, e := io.Copy(&data, r.Response.Body)
	if e != nil {
		r.Error = e
		return
	}
	r.HTML = data.String()
}

// JSONparse - method to return the json content of response
func (r *Res) JSONparse() {
	var data json.RawMessage
	if r.Error != nil {
		return
	}
	defer r.Response.Body.Close()
	e := json.NewDecoder(r.Response.Body).Decode(&data)
	if e != nil {
		r.Error = e
		return
	}
	r.JSON = data
}
