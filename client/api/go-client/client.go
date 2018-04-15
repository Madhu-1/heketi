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
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/heketi/heketi/pkg/utils"
)

const (
	MAX_CONCURRENT_REQUESTS = 1000
	retryCount              = 1000
)

// Client object
type Client struct {
	host       string
	key        string
	user       string
	throttle   chan bool
	retryCount int
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
	defer r.Body.Close()
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

	httpClient := &http.Client{}
	httpClient.CheckRedirect = c.checkRedirect
	return httpClient.Do(req)
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

		// Check if the request is pending
		if r.Header.Get("X-Pending") == "true" {
			if r.StatusCode != http.StatusOK {
				return nil, utils.GetErrorFromResponse(r)
			}
			time.Sleep(waitTime)
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
	r1 := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < c.retryCount; i++ {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(requestBody))
		r, err := c.do(req)
		if err != nil {
			fmt.Println("the error during the request ", err)
			return nil, err
		}
		defer r.Body.Close()
		fmt.Println("status code", r.StatusCode)
		switch r.StatusCode {
		case http.StatusTooManyRequests:

			num := r1.Intn(30)
			fmt.Println("not able to satisfy this request retry in", num)
			time.Sleep(time.Duration(num))
			continue

		default:
			return r, err

		}
	}
	return nil, errors.New("Failed to complete requested operation")
}
