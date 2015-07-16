// Package nface handles all communication between Go code and the Reddit api.
package nface

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/paytonturnage/graw/data"
	"golang.org/x/oauth2"
)

type reqAction int

const (
	// GET describes an http GET request.
	GET = iota
	// POST describes an http POST request.
	POST = iota
)

const (
	// authURL is the url for authorization requests.
	authURL = "https://www.reddit.com/api/v1/access_token"
	// baseURL is the default base url for all api calls.
	baseURL = "https://oauth.reddit.com/api"
	// contentType is a header flag for POST requests so the reddit api
	// knows how to read the request body.
	contentType = "application/x-www-form-urlencoded"
)

// Client manages a connection with the reddit api.
type Client struct {
	// baseURL is the base url for all api calls.
	baseURL string
	// client holds an http.Transport that automatically handles OAuth.
	client *http.Client
	// userAgent is information identifying the graw program to reddit.
	userAgent *data.UserAgent
}

// Request describes how to build an http.Request for the reddit api.
type Request struct {
	// Action is the request type (e.g. "POST" or "GET").
	Action reqAction
	// URL is the url of the api call, which is resolved against the base url.
	URL string
	// Values holds any parameters for the api call; encoded in url.
	Values *url.Values
}

// NewClient returns a new Client struct.
func NewClient(userAgent *data.UserAgent) (*Client, error) {
	client := &Client{baseURL: baseURL, userAgent: userAgent}
	return client, client.oauth(authURL)
}

// TestClient returns an nface.Client which uses the provided http.Client for
// network actions.
func TestClient(c *http.Client, baseURL string) *Client {
	return &Client{baseURL: baseURL, client: c}
}

// Do executes a request using Client's auth and user agent. The result is
// Unmarshal()ed into response.
func (c *Client) Do(r *Request, response interface{}) error {
	req, err := c.buildRequest(r)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}

	return json.Unmarshal(resp, response)
}

// buildRequest builds an http.Request from a Request struct.
func (c *Client) buildRequest(r *Request) (*http.Request, error) {
	var req *http.Request
	var err error

	callURL := fmt.Sprintf("%s%s", c.baseURL, r.URL)
	if r.Action == GET {
		req, err = getRequest(callURL, r.Values)
	} else if r.Action == POST {
		req, err = postRequest(callURL, r.Values)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Add("user-agent", c.userAgent.GetUserAgent())

	return req, nil
}

// doRequest sends a request to the servers and returns the body of the response
// a byte slice.
func (c *Client) doRequest(r *http.Request) ([]byte, error) {
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.Body == nil {
		return nil, fmt.Errorf("empty response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %v\n", resp.StatusCode)
	}

	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %v", err)
	}

	return buf, nil
}

// oauth attempts to authenticate with reddit using OAuth2 and the nface's
// user agent.
func (c *Client) oauth(auth string) error {
	conf := &oauth2.Config{
		ClientID:     c.userAgent.GetClientId(),
		ClientSecret: c.userAgent.GetClientSecret(),
		Endpoint: oauth2.Endpoint{
			TokenURL: auth,
		},
	}

	token, err := conf.PasswordCredentialsToken(
		oauth2.NoContext,
		c.userAgent.GetUsername(),
		c.userAgent.GetPassword())
	if err != nil {
		return err
	}

	c.client = conf.Client(oauth2.NoContext, token)
	return nil
}

// postRequest returns a template http.Request with the given url and POST form
// values set.
func postRequest(url string, vals *url.Values) (*http.Request, error) {
	if vals == nil {
		return nil, fmt.Errorf("no values for POST body")
	}

	reqBody := bytes.NewBufferString(vals.Encode())
	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("content-type", contentType)
	return req, nil
}

// getRequest returns a template http.Request with the given url and GET form
// values set.
func getRequest(url string, vals *url.Values) (*http.Request, error) {
	reqURL := url
	if vals != nil {
		reqURL = fmt.Sprintf("%s?%s", reqURL, vals.Encode())
	}
	return http.NewRequest("GET", reqURL, nil)
}
