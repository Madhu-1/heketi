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

func (reqID *RequestID) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	switch r.Method {

	case http.MethodPost, http.MethodDelete:
		newCtx := context.WithValue(r.Context(), requestIDKey, utils.GenUUID())
		next(w, r.WithContext(newCtx))

	default:
		next(w, r)
	}
}
