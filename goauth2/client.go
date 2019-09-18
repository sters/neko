package goauth2

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/morikuni/failure"
	"github.com/sters/neko/gclient"
)

const (
	oauthURI         = "https://accounts.google.com/o/oauth2/v2/auth"
	authorizationURI = "https://www.googleapis.com/oauth2/v4/token"

	redirectURI                = "urn:ietf:wg:oauth:2.0:oob" // fixed for desktop app
	responseType               = "code"                      // fixed for desktop app
	accessType                 = "offline"                   // fixed for desktop app
	grantTypeAuthorizationCode = "authorization_code"
	grantTypeRefreshToken      = "refresh_token"
)

type (
	AuthorizationResponse struct {
		AccessToken  string `json:"access_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
	}
)

type Client struct {
	c                 *http.Client
	clientID          string
	clientSecret      string
	accessToken       string
	accessTokenExpire int64
	refreshToken      string
	scope             string
}

func (c *Client) GetRefreshToken() string {
	return c.refreshToken
}
func (c *Client) GetAccessToken() string {
	return c.accessToken
}

func NewClient(clientID string, clientSecret string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}
func (c *Client) WithAccessToken(accessToken string) {
	c.accessToken = accessToken
}
func (c *Client) WithScope(scope string) {
	c.scope = scope
}
func (c *Client) WithScopes(scopes ...string) {
	builder := strings.Builder{}
	for _, scope := range scopes {
		builder.WriteString(scope)
		builder.WriteString("&")
	}
	s := builder.String()
	c.scope = s[:len(s)-1]
}
func (c *Client) WithHTTPClient(client *http.Client) {
	c.c = client
}

func (c *Client) GetOAuthURI() string {
	builder := strings.Builder{}
	builder.WriteString(oauthURI)
	builder.WriteString("?")
	builder.WriteString("&client_id=")
	builder.WriteString(c.clientID)
	builder.WriteString("&redirect_uri=")
	builder.WriteString(redirectURI)
	builder.WriteString("&scope=")
	builder.WriteString(c.scope)
	builder.WriteString("&access_type=")
	builder.WriteString(accessType)
	builder.WriteString("&response_type=")
	builder.WriteString(responseType)
	return builder.String()
}

func (c *Client) Authorization(ctx context.Context, authorizationCode string) error {
	params := url.Values{}
	params.Add("code", authorizationCode)
	params.Add("client_id", c.clientID)
	params.Add("client_secret", c.clientSecret)
	params.Add("redirect_uri", redirectURI)
	params.Add("grant_type", grantTypeAuthorizationCode)
	params.Add("access_type", accessType)

	req, err := http.NewRequest(
		http.MethodPost,
		authorizationURI,
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return failure.Wrap(err)
	}

	req = req.WithContext(ctx)
	req.Header.Add(gclient.ContentTypeHeader, gclient.ContentTypeForm)

	rawResponse, err := c.c.Do(req)
	if err != nil {
		return failure.Wrap(err)
	}
	defer rawResponse.Body.Close()

	responseBuf, err := ioutil.ReadAll(rawResponse.Body)
	if err != nil {
		return failure.Wrap(err)
	}

	var response AuthorizationResponse
	err = json.Unmarshal(responseBuf, &response)
	if err != nil {
		log.Println(rawResponse)
		return failure.Wrap(err)
	}

	c.accessToken = response.AccessToken
	c.accessTokenExpire = response.ExpiresIn
	c.refreshToken = response.RefreshToken

	return nil
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) error {
	c.refreshToken = refreshToken

	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("grant_type", grantTypeRefreshToken)
	params.Set("refresh_token", c.refreshToken)

	req, err := http.NewRequest(
		http.MethodPost,
		authorizationURI,
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return failure.Wrap(err)
	}

	req = req.WithContext(ctx)
	req.Header.Add(gclient.ContentTypeHeader, gclient.ContentTypeForm)

	rawResponse, err := c.c.Do(req)
	if err != nil {
		return failure.Wrap(err)
	}
	defer rawResponse.Body.Close()

	responseBuf, err := ioutil.ReadAll(rawResponse.Body)
	if err != nil {
		return failure.Wrap(err)
	}

	var response AuthorizationResponse
	err = json.Unmarshal(responseBuf, &response)
	if err != nil {
		log.Println(rawResponse)
		return failure.Wrap(err)
	}

	if response.AccessToken != "" {
		c.accessToken = response.AccessToken
		c.accessTokenExpire = response.ExpiresIn
	}
	if response.RefreshToken != "" {
		c.refreshToken = response.RefreshToken
	}

	return nil
}
