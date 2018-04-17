//
// Copyright (c) 2015 The heketi Authors
//
// This file is licensed to you under your choice of the GNU Lesser
// General Public License, version 3 or any later version (LGPLv3 or
// later), as published by the Free Software Foundation,
// or under the Apache License, Version 2.0 <LICENSE-APACHE2 or
// http://www.apache.org/licenses/LICENSE-2.0>.
//
// You may not use this file except in compliance with those terms.
//

package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/heketi/heketi/pkg/utils"
)

const (
	MAX_CONCURRENT_REQUESTS = 32
	retryCount              = 1000
)

// Client object
type Client struct {
	host       string
	key        string
	user       string
	throttle   chan bool
	retryCount int
	httpClient *http.Client
}

//NewClient Creates a new client to access a Heketi server
func NewClient(host, user, key string) *Client {
	return NewClientWithRetry(host, user, key, retryCount)
}

//NewClientWithRetry Creates a new client to access a Heketi server with retrycount
func NewClientWithRetry(host, user, key string, retryCount int) *Client {
	c := &Client{}

	c.key = key
	c.host = host
	c.user = user
	//maximum retry for request
	c.retryCount = retryCount
	// Maximum concurrent requests
	c.throttle = make(chan bool, MAX_CONCURRENT_REQUESTS)
	c.httpClient = &http.Client{}
	// transport := http.Transport{
	// 	MaxIdleConnsPerHost: 100,
	// }
	// c.httpClient.Transport = &transport
	c.httpClient.CheckRedirect = c.checkRedirect
	return c
}

// Create a client to access a Heketi server without authentication enabled
func NewClientNoAuth(host string) *Client {
	return NewClient(host, "", "")
}

// Simple Hello test to check if the server is up
func (c *Client) Hello() error {
	// Create request
	req, err := http.NewRequest("GET", c.host+"/hello", nil)
	if err != nil {
		return err
	}

	// Set token
	err = c.setToken(req)
	if err != nil {
		return err
	}

	// Get info
	r, err := c.do(req)
	if err != nil {
		return err
	}
	//defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return utils.GetErrorFromResponse(r)
	}

	return nil
}

// Make sure we do not run out of fds by throttling the requests
func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.throttle <- true
	defer func() {
		<-c.throttle
	}()

	return c.httpClient.Do(req)
}

// This function is called by the http package if it detects that it needs to
// be redirected.  This happens when the server returns a 303 HTTP Status.
// Here we create a new token before it makes the next request.
func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	return c.setToken(req)
}

// Wait for the job to finish, waiting waitTime on every loop
func (c *Client) waitForResponseWithTimer(r *http.Response,
	waitTime time.Duration) (*http.Response, error) {

	// Get temp resource
	location, err := r.Location()
	if err != nil {
		return nil, err
	}

	for {
		// Create request
		req, err := http.NewRequest("GET", location.String(), nil)
		if err != nil {
			return nil, err
		}

		// Set token
		err = c.setToken(req)
		if err != nil {
			return nil, err
		}

		// Wait for response
		r, err = c.do(req)
		if err != nil {
			return nil, err
		}
		//defer r.Body.Close()
		// Check if the request is pending
		if r.Header.Get("X-Pending") == "true" {
			if r.StatusCode != http.StatusOK {
				return nil, utils.GetErrorFromResponse(r)
			}
			time.Sleep(waitTime)
			// if r != nil {
			// 	//close connection once the request is served
			// 	//to solve to many file descriptors open issue
			// 	io.Copy(ioutil.Discard, r.Body)
			// 	r.Body.Close()
			// }

		} else {
			return r, nil
		}
	}

}

// Create JSON Web Token
func (c *Client) setToken(r *http.Request) error {

	// Create qsh hash
	qshstring := r.Method + "&" + r.URL.Path
	hash := sha256.New()
	hash.Write([]byte(qshstring))

	// Create Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		// Set issuer
		"iss": c.user,

		// Set issued at time
		"iat": time.Now().Unix(),

		// Set expiration
		"exp": time.Now().Add(time.Minute * 5).Unix(),

		// Set qsh
		"qsh": hex.EncodeToString(hash.Sum(nil)),
	})

	// Sign the token
	signedtoken, err := token.SignedString([]byte(c.key))
	if err != nil {
		return err
	}

	// Save it in the header
	r.Header.Set("Authorization", "bearer "+signedtoken)

	return nil
}

//RetryOperationDo for retry operation
func (c *Client) retryOperationDo(req *http.Request, requestBody []byte) (*http.Response, error) {
	// Send request
	for i := 0; i < c.retryCount; i++ {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(requestBody))

		r, err := c.do(req)
		if err != nil {
			return nil, err
		}
		// if r != nil {
		// 	//close connection once the request is served
		// 	//to solve to many file descriptors open issue
		// 	io.Copy(ioutil.Discard, r.Body)
		// 	r.Body.Close()
		// }
		//defer r.Body.Close()
		switch r.StatusCode {
		case http.StatusTooManyRequests:
			if r != nil {
				//close connection once the request is served
				//to solve to many file descriptors open issue
				io.Copy(ioutil.Discard, r.Body)
				r.Body.Close()
			}
			num := random(10, 40)
			time.Sleep(time.Second * time.Duration(num))
			continue

		default:
			return r, err

		}
	}
	return nil, errors.New("Failed to complete requested operation")
}

func random(min, max int) int {
	return rand.Intn(max-min) + min
}
