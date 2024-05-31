package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"github.com/libp2p/go-libp2p/core/peer"
)

type NoRedirectError struct{}

func (NoRedirectError) Error() string {
	return "no redirect"
}

func NoRedirect(req *http.Request, via []*http.Request) error {
	return NoRedirectError{}
}

type Payload struct {
	Hash         string  `json:"hash"`
	PeerIdentity peer.ID `json:"identity"`
}

func NewPayload(r io.Reader) (*Payload, error) {
	var payload Payload
	return &payload, payload.FromReader(r)
}

func (p *Payload) FromReader(r io.Reader) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(p)
}

func (p Payload) Reader() (io.Reader, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	err := encoder.Encode(p)
	return &buf, err
}

type UserLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (u UserLogin) Reader() (io.Reader, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	err := encoder.Encode(u)
	return &buf, err
}

type GeneralResponse struct {
	Message string
	Error   string
}

func NewGeneralResponse(r io.Reader) (GeneralResponse, error) {
	var g GeneralResponse
	return g, g.FromReader(r)
}

func (g *GeneralResponse) FromReader(r io.Reader) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(g)
}

type GetCredentialsHandler func() (UserLogin, error)
type WriteCookiesHandler func() error

type Client struct {
	*http.Client
	baseUrl        *url.URL
	GetCredentials GetCredentialsHandler
	CookieFilename string
}

func NewClient(baseUrl *url.URL) (*Client, error) {
	if baseUrl == nil {
		return nil, fmt.Errorf("cannot create client: invalid url: %v", baseUrl)
	}

	// TODO: Setup PublicSuffixList
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &Client{
		Client: &http.Client{
			CheckRedirect: NoRedirect,
			Jar:           jar,
		},
		baseUrl:        baseUrl,
		GetCredentials: DefaultGetCredentials,
	}
	return client, nil
}

func NewClientWithCookies(baseUrl *url.URL, cookieFilename string) (*Client, error) {
	client, err := NewClient(baseUrl)
	if err != nil {
		return client, err
	}

	if cookieFilename != "" {
		client.CookieFilename = cookieFilename
	}

	return client, client.ReadCookies(client.CookieFilename)
}

func (c *Client) Cookies() []*http.Cookie {
	return c.Jar.Cookies(c.baseUrl)
}

func cleanup(filename string, file *os.File) {
	file.Close()
	if err := os.Chmod(filename, 0700); err != nil {
		os.Remove(filename)
	}
}

func (c *Client) ReadCookies(filename string) error {
	fmt.Println("ReadCookies")
	if c == nil {
		return fmt.Errorf("client is not initialized")
	}

	handle, err := os.Open(filename)
	defer cleanup(filename, handle)

	if err != nil {
		return nil
	}

	decoder := json.NewDecoder(handle)
	var cookies []*http.Cookie
	if err := decoder.Decode(&cookies); err != nil {
		return err
	}

	c.Jar.SetCookies(c.baseUrl, cookies)

	return nil
}

func (c *Client) WriteCookies(filename string) error {
	if c == nil {
		return fmt.Errorf("client is not initialized")
	}

	cookies := c.Cookies()
	if len(cookies) == 0 {
		return nil
	}

	handle, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
	defer func() {
		handle.Close()
		if err := os.Chmod(filename, 0700); err != nil {
			os.Remove(filename)
		}
	}()
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(handle)
	return encoder.Encode(cookies)
}

type Result struct {
	RequiredLogin bool
	Payload       *Payload
	Code          int
	Error         error
}

func (c *Client) Login(creds UserLogin) (result Result) {
	fmt.Println("Logging in")

	if c == nil {
		result.Error = fmt.Errorf("client is not initialized")
		return
	}

	r, err := creds.Reader()
	if err != nil {
		result.Error = fmt.Errorf("failed to create login payload")
		return
	}

	endpoint := c.baseUrl.JoinPath("login").String()

	req, err := http.NewRequest("POST", endpoint, r)
	if err != nil {
		result.Error = err
		return
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.Do(req)
	if resp != nil {
		result.Code = resp.StatusCode
	}

	if err != nil {
		result.Error = err
		return
	} else if result.Code != 200 {
		result.Error = fmt.Errorf(http.StatusText(result.Code))
		return
	}

	g, err := NewGeneralResponse(resp.Body)
	if err != nil {
		result.Error = err
		return
	}

	if g.Error != "" {
		result.Error = fmt.Errorf("login failed: %s", g.Error)
		return
	}

	return
}

func (c *Client) sendRequest(req *http.Request) (result Result) {
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.Do(req)
	if resp != nil {
		result.Code = resp.StatusCode
	}

	if err != nil {
		result.Error = err
		return
	} else if result.Code != 200 {
		result.Error = fmt.Errorf(http.StatusText(result.Code))
		return
	}

	result.Payload, err = NewPayload(resp.Body)
	if err != nil {
		result.Error = err
		return
	}

	return
}

func (c *Client) GetAsset(path ...string) Result {
	if c == nil {
		return Result{
			Error: fmt.Errorf("client is not initialized"),
		}
	}

	endpoint := c.baseUrl.JoinPath(path...).String()

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return Result{
			Error: err,
		}
	}

	return c.sendRequest(req)
}

func (c *Client) PostAsset(payload Payload, path ...string) (result Result) {
	if c == nil {
		return Result{
			Error: fmt.Errorf("client is not initialized"),
		}
	}

	r, err := payload.Reader()
	if err != nil {
		return Result{
			Error: err,
		}
	}

	endpoint := c.baseUrl.JoinPath(path...).String()

	req, err := http.NewRequest("POST", endpoint, r)
	if err != nil {
		return Result{
			Error: err,
		}
	}

	return c.sendRequest(req)
}

type SendRequestHandler func() Result

func (c *Client) LoginGuard(req SendRequestHandler) Result {
	if c == nil {
		return Result{
			Error: fmt.Errorf("client is not initialized"),
		}
	}

	if c.GetCredentials == nil {
		return Result{
			Error: fmt.Errorf("client has no GetCredentialsHandler"),
		}
	}

	result := req()

	if result.Code == http.StatusUnauthorized {
		creds, err := c.GetCredentials()
		if err != nil {
			return Result{
				RequiredLogin: true,
				Error:         err,
			}
		}

		if result = c.Login(creds); result.Error != nil {
			return Result{
				RequiredLogin: true,
				Error:         result.Error,
			}
		}

		result = req()
		result.RequiredLogin = true
	}

	if c.CookieFilename != "" {
		c.WriteCookies(c.CookieFilename)
	}

	return result
}
