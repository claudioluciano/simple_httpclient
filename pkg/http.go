package http

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type HTTPMethod string

const (
	POST   HTTPMethod = "POST"
	GET    HTTPMethod = "GET"
	PUT    HTTPMethod = "PUT"
	PATH   HTTPMethod = "POST"
	DELETE HTTPMethod = "DELETE"
)

type Request struct {
	URL         string
	Method      HTTPMethod
	ContentType string
	Headers     map[string]string
	Query       map[string]string
	Body        string
}

type Response struct {
	Body       string
	Headers    map[string]string
	StatusCode int32
	Time       int64
}

type Client struct {
	Client      *fasthttp.Client
	baseURL     string
	contentType string
	timeout     time.Duration
}

type NewClientOptions struct {
	BaseURL            string
	DefaultContentType string
	Timeout            time.Duration
	Attemps            int
	TLSCert            string
}

type DoOptions struct {
	StartTime *time.Time
	Request   *Request
}

func New(opts ...*NewClientOptions) *Client {
	opt := &NewClientOptions{
		DefaultContentType: "application/json",
		Timeout:            30 * time.Second,
		Attemps:            5,
	}

	if len(opts) > 0 {
		opt = opts[0]
	}

	c := &Client{
		Client:      &fasthttp.Client{},
		baseURL:     opt.BaseURL,
		contentType: opt.DefaultContentType,
		timeout:     opt.Timeout,
	}

	c.Client.MaxIdemponentCallAttempts = opt.Attemps

	return c
}

func (h *Client) Do(ctx context.Context, opts *DoOptions) (*Response, error) {
	if opts.StartTime == nil {
		n := time.Now()
		opts.StartTime = &n
	}

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	u := h.getURL(opts.Request.URL)

	qp := url.Values{}
	for k, v := range opts.Request.Query {
		qp.Add(k, fmt.Sprint(v))
	}

	if len(qp) > 0 {
		u = fmt.Sprintf("%s?%s", u, qp.Encode())
	}

	cType := h.contentType
	if opts.Request.ContentType != "" {
		cType = opts.Request.ContentType
	}

	req.SetRequestURI(u)
	req.Header.SetMethod(string(opts.Request.Method))
	req.Header.SetContentType(cType)
	req.SetBodyString(opts.Request.Body)

	for k, v := range opts.Request.Headers {
		req.Header.Set(k, v)
	}

	if err := h.Client.DoTimeout(req, res, h.timeout); err != nil {
		return nil, err
	}

	endNow := time.Now()

	return &Response{
		StatusCode: int32(res.StatusCode()),
		Body:       string(res.Body()),
		Headers:    mergeResponseHeaders(&res.Header),
		Time:       getResponseTime(opts.StartTime, &endNow),
	}, nil
}

func (h *Client) getURL(rURL string) string {
	if strings.HasPrefix(rURL, "http") || strings.HasPrefix(rURL, "https") {
		return rURL
	}

	return h.baseURL + rURL
}

func mergeResponseHeaders(h *fasthttp.ResponseHeader) map[string]string {
	headers := map[string]string{}

	h.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	return headers
}

func getResponseTime(start, end *time.Time) int64 {
	return end.Sub(*start).Milliseconds()
}
