package velux

import (
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// https://community.openhab.org/t/connecting-velux-active-kix-300/75696/41
const clientID = "5931426da127d981e76bdd3f"
const clientSecret = "6ae2d89d15e767ae5c56b456b452d319"

var Endpoint = oauth2.Endpoint{
	AuthURL:   "https://app.velux-active.com/oauth2/token",
	TokenURL:  "https://app.velux-active.com/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

// AuthTransport can be used to add the required custom fields for an
// OAuth2 client using golang.org/x/oauth2.
//
// Example:
// hc := &http.Client{Transport: &AuthTransport{VG: *vg}}
// ctx := context.WithValue(context.Background(), oauth2.HTTPClient, hc)
type AuthTransport struct {
	AppVersion string `json:"app_version"`
	UserPrefix string `json:"user_prefix"`
}

func DefaultAuthTransport() *AuthTransport {
	return &AuthTransport{
		AppVersion: "1108002",
		UserPrefix: "velux",
	}
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}

	vals.Set("app_version", t.AppVersion)
	vals.Set("user_prefix", t.UserPrefix)

	buf := strings.NewReader(vals.Encode())
	req.Body = ioutil.NopCloser(buf)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
	req.ContentLength = int64(buf.Len())

	// Call default roundtrip
	return http.DefaultTransport.RoundTrip(req)
}
