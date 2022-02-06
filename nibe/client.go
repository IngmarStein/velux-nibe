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

// NewClientWithAuth returns a new NIBE API client using the supplied credentials.
func NewClientWithAuth(clientID, clientSecret, callbackURL, tokenFileName string, scopes []string) *Client {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     Endpoint,
		RedirectURL:  callbackURL,
		Scopes:       scopes,
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

type ImageSize struct {
	// Width
	Width int `json:"width"`
	// Height
	Height int `json:"height"`
	// URL. Default URI-scheme and hostname when left out are https://www.nibeuplink.com
	URL string `json:"url"`
}

type Image struct {
	// Unique name for image
	Name string `json:"name"`
	// List of image sizes available
	Sizes []ImageSize `json:"sizes"`
}

type Parameter struct {
	// Parameter id.
	ParameterID int `json:"parameterId"`
	// Name used for parameter in the request
	Name string `json:"name"`
	// Parameter title
	Title string `json:"title"`
	// Parameter designation
	Designation string `json:"designation"`
	// Unit
	Unit string `json:"unit"`
	// Human readable representation of the raw value
	DisplayValue string `json:"displayValue"`
	// Raw value, as handled by the system itself
	RawValue int `json:"rawValue"`
}

type StatusIconItem struct {
	// Image
	Image Image `json:"image"`
	// Text displayed on top of / close to image to help describe the picture.
	// E.g. for a compressor it could contain the name of the compressor module.
	InlineText string `json:"inlineText"`
	// Title
	Title string `json:"title"`
	// List of parameters closely related to this item
	Parameters []Parameter `json:"parameters"`
}

type Category struct {
	CategoryID string      `json:"categoryId"`
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
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

type GetSystemStatusRequest struct {
	SystemID int
}

type GetSystemStatusResponse []StatusIconItem

type GetSystemParametersRequest struct {
	SystemID     int
	ParameterIDs []string
}

type GetSystemParametersResponse []Parameter

type GetServiceInfoCategoriesRequest struct {
	SystemID     int
	SystemUnitID int
	Parameters   bool
}

type GetServiceInfoCategoriesResponse []Category

// SetThermostat upload thermostat data to NIBE Uplink.
// Use the ExternalId parameter to identify which thermostat to update, if it
// does not already exist a thermostat with the supplied id will be created.
// Even though no change may have occured the thermostat needs to report its
// current status at least every 30 minutes to continue affecting the system.
func (c *Client) SetThermostat(request SetThermostatRequest) error {
	u := fmt.Sprintf("systems/%d/smarthome/thermostats", request.SystemID)

	req, err := c.NewRequest("POST", u, request)
	if err != nil {
		return err
	}
	_, err = c.do(req, nil)
	return err
}

// GetSystemStatus returns the current overall system status.
// This includes which system components (e.g. additional heating and
// accessories) are currently running as well as what the system is currently
// producing (e.g. hot water and heating). For systems with only one system
// unit containing a compressor module the function also returns whether any
// compressors or pumps belonging to this system unit is currently running.
// For larger systems you need to check each individual system unit's status
// to get their compressor and pump status.
func (c *Client) GetSystemStatus(request GetSystemStatusRequest) (GetSystemStatusResponse, error) {
	u := fmt.Sprintf("systems/%d/status/system", request.SystemID)

	req, err := c.NewRequest("GET", u, request)
	if err != nil {
		return nil, err
	}
	var response GetSystemStatusResponse
	_, err = c.do(req, &response)
	return response, err
}

func (c *Client) GetSystemParameters(request GetSystemParametersRequest) (GetSystemParametersResponse, error) {
	u := fmt.Sprintf("systems/%d/parameters", request.SystemID)

	req, err := c.NewRequest("GET", u, request)
	if err != nil {
		return nil, err
	}
	var response GetSystemParametersResponse
	_, err = c.do(req, &response)
	return response, err
}

func (c *Client) GetServiceInfoCategories(request GetServiceInfoCategoriesRequest) (GetServiceInfoCategoriesResponse, error) {
	includeParameters := "false"
	if request.Parameters {
		includeParameters = "true"
	}
	u := fmt.Sprintf("systems/%d/serviceinfo/categories?systemUnitId=%d&parameters=%s", request.SystemID, request.SystemUnitID, includeParameters)

	req, err := c.NewRequest("GET", u, request)
	if err != nil {
		return nil, err
	}
	var response GetServiceInfoCategoriesResponse
	_, err = c.do(req, &response)
	return response, err
}
