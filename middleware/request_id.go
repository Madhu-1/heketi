//
// Copyright (c) 2018 The heketi Authors
//
// This file is licensed to you under your choice of the GNU Lesser
// General Public License, version 3 or any later version (LGPLv3 or
// later), or the GNU General Public License, version 2 (GPLv2), in all
// cases as published by the Free Software Foundation.
//

package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/heketi/heketi/pkg/utils"
)

type contextKey string

var requestIDKey = contextKey("X-Request-ID")

type RequestID struct {
}

// GetRequestID returns the request id from HTTP context.
func GetRequestID(ctx context.Context) string {
	reqID, _ := ctx.Value(requestIDKey).(string)
	return reqID
}
func AddRequestID(r *http.Request) context.Context {
	return context.WithValue(r.Context(), requestIDKey, utils.GenUUID())
}
func (reqID *RequestID) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	switch r.Method {

	case http.MethodPost, http.MethodDelete:
		newCtx := AddRequestID(r)
		next(w, r.WithContext(newCtx))

	case http.MethodGet:
		path := strings.TrimRight(r.URL.Path, "/")
		urlPart := strings.Split(path, "/")
		if len(urlPart) >= 4 {
			if urlPart[1] == "devices" && urlPart[3] == "resync" {
				newCtx := AddRequestID(r)
				next(w, r.WithContext(newCtx))
				return
			}
		}

		next(w, r)

	default:
		next(w, r)
	}
}
