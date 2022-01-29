package velux

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strings"

	"github.com/google/go-querystring/query"
)

const (
	defaultBaseURL = "https://app.velux-active.com/api/"
	userAgent      = "go-velux"
)

type Client struct {
	BaseURL   *url.URL
	UserAgent string
	Verbose   bool

	client *http.Client
}

// NewClientWithAuth returns a new Velux API client using the supplied credentials.
func NewClientWithAuth(username, password string) *Client {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     Endpoint,
	}

	hc := &http.Client{Transport: DefaultAuthTransport()}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, hc)

	token, err := conf.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		log.Fatalf("error retrieving token: %v", err)
	}

	oauthClient := conf.Client(context.Background(), token)
	return NewClient(oauthClient)
}

// NewClient returns a new Velux API client. If a nil httpClient is
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

// addOptions adds the parameters in opt as URL query parameters to s. opt
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

const RollerShutter = "NXO"
const Bridge = "NXG"
const DepartureSwitch = "NXD"
const Sensor = "NXS"

type GetHomesDataRequest struct {
	GatewayTypes []string `url:"gateway_types,omitempty"`
}

type GetHomesDataResponse struct {
	Body struct {
		Homes []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Rooms []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"rooms"`
		} `json:"homes"`
	} `json:"body"`
}

func (c *Client) GetHomesData(request GetHomesDataRequest) (GetHomesDataResponse, error) {
	u, err := addOptions("gethomesdata", request)
	if err != nil {
		return GetHomesDataResponse{}, err
	}

	req, err := c.NewRequest("GET", u, nil)
	if err != nil {
		return GetHomesDataResponse{}, err
	}
	var response GetHomesDataResponse
	_, err = c.do(req, &response)
	return response, err
}

type HomeStatusRequest struct {
	HomeID      string   `url:"home_id"`
	DeviceTypes []string `url:"device_types"`
}

type HomeStatusResponse struct {
	Body struct {
		Home struct {
			ID    string `json:"id"`
			Rooms []struct {
				AirQuality            int    `json:"air_quality"`
				AlgoScheduleStart     int    `json:"algo_schedule_start"`
				AlgoStatus            int    `json:"algo_status"`
				AutoCloseTS           int    `json:"auto_close_ts"`
				CO2                   int    `json:"co2"`
				Humidity              int    `json:"humidity"`
				ID                    string `json:"id"`
				Lux                   int    `json:"lux"`
				MaxComfortCO2         int    `json:"max_comfort_co2"`
				MaxComfortHumidity    int    `json:"max_comfort_humidity"`
				MaxComfortTemperature int    `json:"max_comfort_temperature"`
				MinComfortHumidity    int    `json:"min_comfort_humidity"`
				MinComfortTemperature int    `json:"min_comfort_temperature"`
				Temperature           int    `json:"temperature"`
			} `json:"rooms"`
		} `json:"home"`
	} `json:"body"`
}

func (c *Client) HomeStatus(request HomeStatusRequest) (HomeStatusResponse, error) {
	u, err := addOptions("homestatus", request)
	if err != nil {
		return HomeStatusResponse{}, err
	}

	req, err := c.NewRequest("GET", u, nil)
	if err != nil {
		return HomeStatusResponse{}, err
	}
	var response HomeStatusResponse
	_, err = c.do(req, &response)
	return response, err
}
