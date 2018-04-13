//Package middleware for heketi
// Copyright (c) 2018 The heketi Authors
//
// This file is licensed to you under your choice of the GNU Lesser
// General Public License, version 3 or any later version (LGPLv3 or
// later), or the GNU General Public License, version 2 (GPLv2), in all
// cases as published by the Free Software Foundation.
//
package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/heketi/utils"
	"github.com/urfave/negroni"
)

var (
	throttleNow func() time.Time = time.Now
	logger                       = utils.NewLogger("[heketi]", utils.LEVEL_INFO)
)

//ReqLimiter struct holds data related to Throttling
type ReqLimiter struct {
	maxcount     uint32
	servingCount uint32
	//counter for incoming request
	reqRecvCount uint32
	//in memeory storage for ReqLimiter
	requestCache map[string]time.Time
	lock         sync.RWMutex
	stop         chan<- interface{}
}

//Function to check can heketi can take more request
func (r *ReqLimiter) reachedMaxRequest() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.servingCount >= r.maxcount
}

//Function to check total received request
func (r *ReqLimiter) reqReceivedcount() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.reqRecvCount >= r.maxcount
}

//Function to add request id to the queue
func (r *ReqLimiter) incRequest(reqid string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.requestCache[reqid] = throttleNow()
	r.servingCount++
}

//Function to remove request id to the queue
func (r *ReqLimiter) decRequest(reqid string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.requestCache, reqid)
	r.servingCount--

}

//Function to increment recvRequestCount
func (r *ReqLimiter) incRecvCount() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reqRecvCount++
}

//Function to decrement recvRequestCount
func (r *ReqLimiter) decRecvCount() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reqRecvCount--

}

var tr *ReqLimiter

//NewHTTPThrottler Function to return the ReqLimiter
func NewHTTPThrottler(count uint32) *ReqLimiter {
	tr = &ReqLimiter{
		maxcount:     count,
		requestCache: make(map[string]time.Time),
	}
	return &ReqLimiter{
		maxcount:     count,
		requestCache: make(map[string]time.Time),
	}

}

func (r *ReqLimiter) ServeHTTP(hw http.ResponseWriter, hr *http.Request, next http.HandlerFunc) {

	switch hr.Method {

	case http.MethodPost, http.MethodDelete:
		//recevied a request increment the counter
		//this is required it will take time to get response for current request

		tr.incRecvCount()
		//by this we can avoid overload by checking maximum and currently received requests counts
		logger.Info("madhu serving request and received request count %v %v", tr.servingCount, tr.reqRecvCount)
		if !tr.reachedMaxRequest() && !tr.reqReceivedcount() {

			next(hw, hr)

			res, ok := hw.(negroni.ResponseWriter)
			if !ok {
				return
			}
			//if request is accepted for Async operation
			logger.Info("madhu request accepted status and id %v %v", res.Status(), GetRequestID(hr.Context()))
			if res.Status() == http.StatusAccepted {

				reqID := GetRequestID(hr.Context())
				//Add request Id to in-memory
				if reqID != "" {

					tr.incRequest(reqID)
					logger.Info(fmt.Sprintf("serving request count %v", tr.servingCount))
				}

			}
		} else {
			logger.LogError(fmt.Sprintf("Rejected the request for URL max count reached recvcount %v serving count %v", tr.reqRecvCount, tr.servingCount))
			http.Error(hw, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)

		}
		tr.decRecvCount()
	case http.MethodGet:

		next(hw, hr)

		res, ok := hw.(negroni.ResponseWriter)
		if !ok {
			return
		}
		path := strings.TrimRight(hr.URL.Path, "/")
		urlPart := strings.Split(path, "/")
		if len(urlPart) >= 3 {
			if isSuccess(res.Status()) || res.Status() == http.StatusInternalServerError {
				//extract the reqID from URL
				reqID := urlPart[2]
				//check Request Id present in im-memeory
				if _, ok := tr.requestCache[reqID]; ok {
					//check operation is not pending
					if hr.Header.Get("X-Pending") != "true" {

						tr.decRequest(reqID)
						logger.Info("madhu completed for", reqID, tr.servingCount)
					}

				}
			}
		}

	default:
		next(hw, hr)
	}
	return
}

//Cleanup up function to remove stale reqID
func (r *ReqLimiter) Cleanup(ct time.Duration) {
	t := time.NewTicker(ct)
	stop := make(chan interface{})
	tr.stop = stop

	defer t.Stop()
	for {
		select {
		case <-stop:
			return

		case <-t.C:
			tr.lock.Lock()
			for reqID, value := range tr.requestCache {

				//using time.Now() which is helps for testing
				if time.Now().Sub(value) > ct {
					delete(tr.requestCache, reqID)
					tr.servingCount--
				}
			}
			tr.lock.Unlock()
		}
	}
}

//Stop Cleanup
func (r *ReqLimiter) Stop() {
	tr.stop <- true
}

// To check success status code
func isSuccess(status int) bool {

	if status >= http.StatusOK && status < http.StatusResetContent {
		return true
	}
	return false
}
