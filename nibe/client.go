package nibe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	defaultBaseURL = "https://api.nibeuplink.com/api/v1/"
	userAgent      = "go-nibe"
)

type Client struct {
	BaseURL   *url.URL
	UserAgent string
	Verbose   bool

	client *http.Client
}

// NewClient returns a new NIBE API client using the supplied credentials.
func NewClientWithAuth(clientID, clientSecret, callbackURL, tokenFileName string) *Client {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     Endpoint,
		RedirectURL:  callbackURL,
		Scopes:       []string{ScopeWrite},
	}

	oauthClient := GetAuthClient(conf, tokenFileName)
	return NewClient(oauthClient)
}

// NewClient returns a new NIBE API client. If a nil httpClient is
// provided, a new http.Client will be used. To use API methods which require
// authentication, provide an http.Client that will perform the authentication
// for you (such as that provided by the golang.org/x/oauth2 library).
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	baseURL, _ := url.Parse(defaultBaseURL)

	c := &Client{client: httpClient, BaseURL: baseURL, UserAgent: userAgent}

	return c
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified without a preceding slash. If
// specified, the value pointed to by body is JSON encoded and included as the
// request body.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	if c.Verbose {
		if d, err := httputil.DumpRequest(req, true); err == nil {
			log.Println(string(d))
		}
	}

	resp, err := c.client.Do(req)

	if c.Verbose {
		if d, err := httputil.DumpResponse(resp, true); err == nil {
			log.Println(string(d))
		}
	}

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		err = json.NewDecoder(resp.Body).Decode(v)
	}
	return resp, err
}

type SetThermostatRequest struct {
	SystemID int
	// 	Id number set by the smart home system
	ExternalId int `json:"externalId"`
	// Human readable name for the thermostat
	Name string `json:"name"`
	// 	Optional, actual temperature in deg. Celsius, multiplied by 10.
	ActualTemp int `json:"actualTemp"`
	// 	Optional, target temperature in deg. Celsius, multiplied by 10.
	TargetTemp int `json:"targetTemp"`
	// 	Optional, valve position. Number of percent open.
	ValvePosition int `json:"valvePosition"`
	// 	Optional, list of climate systems this thermostat affects.
	ClimateSystems []int `json:"climateSystems"`
}

func (c *Client) SetThermostat(request SetThermostatRequest) error {
	u := fmt.Sprintf("systems/%d/smarthome/thermostats", request.SystemID)

	req, err := c.NewRequest("POST", u, request)
	if err != nil {
		return err
	}
	_, err = c.do(req, nil)
	return err
}
