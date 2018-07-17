//
// Copyright (c) 2014 MessageBird B.V.
// All rights reserved.
//
// Author: Maurice Nonnekes <maurice@messagebird.com>

// Package messagebird is an official library for interacting with MessageBird.com API.
// The MessageBird API connects your website or application to operators around the world. With our API you can integrate SMS, Chat & Voice.
// More documentation you can find on the MessageBird developers portal: https://developers.messagebird.com/
package messagebird

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
)

const (
	// ClientVersion is used in User-Agent request header to provide server with API level.
	ClientVersion = "4.2.1"

	// Endpoint points you to MessageBird REST API.
	Endpoint = "https://rest.messagebird.com"

	// httpClientTimeout is used to limit http.Client waiting time.
	httpClientTimeout = 15 * time.Second
)

var (
	// ErrUnexpectedResponse is used when there was an internal server error and nothing can be done at this point.
	ErrUnexpectedResponse = errors.New("The MessageBird API is currently unavailable")
)

// Client is used to access API with a given key.
// Uses standard lib HTTP client internally, so should be reused instead of created as needed and it is safe for concurrent use.
type Client struct {
	AccessKey  string       // The API access key
	HTTPClient *http.Client // The HTTP client to send requests on
	DebugLog   *log.Logger  // Optional logger for debugging purposes
}

// New creates a new MessageBird client object.
func New(accessKey string) *Client {
	return &Client{
		AccessKey: accessKey,
		HTTPClient: &http.Client{
			Timeout: httpClientTimeout,
		},
	}
}

// Request is for internal use only and unstable.
func (c *Client) Request(v interface{}, method, path string, data interface{}) error {
	if !strings.HasPrefix(path, "https://") && !strings.HasPrefix(path, "http://") {
		path = fmt.Sprintf("%s/%s", Endpoint, path)
	}
	uri, err := url.Parse(path)
	if err != nil {
		return err
	}

	var jsonEncoded []byte
	if data != nil {
		jsonEncoded, err = json.Marshal(data)
		if err != nil {
			return err
		}
	}

	request, err := http.NewRequest(method, uri.String(), bytes.NewBuffer(jsonEncoded))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "AccessKey "+c.AccessKey)
	request.Header.Set("User-Agent", "MessageBird/ApiClient/"+ClientVersion+" Go/"+runtime.Version())

	if c.DebugLog != nil {
		if data != nil {
			c.DebugLog.Printf("HTTP REQUEST: %s %s %s", method, uri.String(), jsonEncoded)
		} else {
			c.DebugLog.Printf("HTTP REQUEST: %s %s", method, uri.String())
		}
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if c.DebugLog != nil {
		c.DebugLog.Printf("HTTP RESPONSE: %s", string(responseBody))
	}

	// Status code 500 is a server error and means nothing can be done at this
	// point.
	if response.StatusCode == 500 {
		return ErrUnexpectedResponse
	}
	// Status codes 200 and 201 are indicative of being able to convert the
	// response body to the struct that was specified.
	if response.StatusCode == 200 || response.StatusCode == 201 {
		if err := json.Unmarshal(responseBody, &v); err != nil {
			return fmt.Errorf("could not decode response JSON, %s: %v", string(responseBody), err)
		}
		return nil
	}

	// Anything else than a 200/201/500 should be a JSON error.
	var errorResponse ErrorResponse
	if err := json.Unmarshal(responseBody, &errorResponse); err != nil {
		return err
	}

	return errorResponse
}
