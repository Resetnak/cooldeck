// Package coolify implements the Coolify REST API and app.Service.
package coolify

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/credentials"
	"github.com/resetnak/cooldeck/internal/domain"
)

const (
	defaultTimeout = 20 * time.Second
	maxResponse    = 8 << 20
	maxReadRetries = 2
)

type client struct {
	baseURL *url.URL
	token   credentials.Token
	http    *http.Client
}

type clientOptions struct {
	baseURL    string
	token      credentials.Token
	insecure   bool
	httpClient *http.Client
}

func newClient(opts clientOptions) (*client, error) {
	base, err := config.NormalizeBaseURL(opts.baseURL)
	if err != nil {
		return nil, fmt.Errorf("coolify URL: %w", err)
	}
	u, err := url.Parse(strings.TrimSuffix(base, "/") + "/api/v1/")
	if err != nil {
		return nil, fmt.Errorf("build API URL: %w", err)
	}
	if opts.token.IsZero() {
		return nil, errors.New("coolify API token is empty")
	}

	httpClient := opts.httpClient
	if httpClient == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if opts.insecure {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicit per-instance opt-in
		}
		httpClient = &http.Client{Transport: transport, Timeout: defaultTimeout}
	}

	return &client{baseURL: u, token: opts.token, http: httpClient}, nil
}

func (c *client) getJSON(ctx context.Context, path string, query url.Values, dst any) error {
	body, err := c.do(ctx, http.MethodGet, path, query)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return domain.NewError(domain.ErrorDecode, err)
	}
	return nil
}

func (c *client) postJSON(ctx context.Context, path string, query url.Values, dst any) error {
	body, err := c.do(ctx, http.MethodPost, path, query)
	if err != nil {
		return err
	}
	if len(body) == 0 || dst == nil {
		return nil
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return domain.NewError(domain.ErrorDecode, err)
	}
	return nil
}

func (c *client) getText(ctx context.Context, path string) (string, error) {
	body, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *client) do(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	requestURL := c.baseURL.ResolveReference(&url.URL{Path: strings.TrimPrefix(path, "/")})
	requestURL.RawQuery = query.Encode()

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
		if err != nil {
			return nil, domain.NewError(domain.ErrorConfig, err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.token.Secret())
		req.Header.Set("User-Agent", "cooldeck")

		resp, err := c.http.Do(req)
		if err != nil {
			classified := classifyTransportError(ctx, err)
			if method == http.MethodGet && attempt < maxReadRetries && classified.Retryable {
				if waitErr := waitForRetry(ctx, attempt, 0); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, classified
		}

		body, readErr := readResponse(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, domain.NewError(domain.ErrorDecode, readErr)
		}
		if closeErr != nil {
			return nil, domain.NewError(domain.ErrorNetwork, closeErr)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}

		apiErr := classifyHTTPError(resp.StatusCode, resp.Header.Get("Retry-After"))
		canRetry := method == http.MethodGet && attempt < maxReadRetries && apiErr.Retryable
		if !canRetry {
			return nil, apiErr
		}
		if err := waitForRetry(ctx, attempt, apiErr.RetryAfter); err != nil {
			return nil, err
		}
	}
}

func readResponse(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, maxResponse+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(data) > maxResponse {
		return nil, fmt.Errorf("response exceeds %d bytes", maxResponse)
	}
	return data, nil
}

func classifyTransportError(ctx context.Context, err error) *domain.Error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return domain.NewError(domain.ErrorCancelled, ctx.Err())
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return domain.NewError(domain.ErrorTimeout, ctx.Err())
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return domain.NewError(domain.ErrorTimeout, err)
	}
	return domain.NewError(domain.ErrorNetwork, err)
}

func classifyHTTPError(status int, retryAfter string) *domain.Error {
	kind := domain.ErrorUnknown
	switch {
	case status == http.StatusUnauthorized:
		kind = domain.ErrorUnauthorized
	case status == http.StatusForbidden:
		kind = domain.ErrorForbidden
	case status == http.StatusNotFound:
		kind = domain.ErrorNotFound
	case status == http.StatusConflict:
		kind = domain.ErrorConflict
	case status == http.StatusUnprocessableEntity || status == http.StatusBadRequest:
		kind = domain.ErrorValidation
	case status == http.StatusTooManyRequests:
		kind = domain.ErrorRateLimited
	case status >= 500:
		kind = domain.ErrorServer
	}
	err := domain.NewError(kind, nil)
	err.StatusCode = status
	err.RetryAfter = parseRetryAfter(retryAfter, time.Now())
	return err
}

func parseRetryAfter(raw string, now time.Time) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(raw)
	if err != nil || !when.After(now) {
		return 0
	}
	return when.Sub(now)
}

func waitForRetry(ctx context.Context, attempt int, retryAfter time.Duration) error {
	delay := retryAfter
	if delay <= 0 {
		delay = time.Duration(1<<attempt) * 100 * time.Millisecond
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return classifyTransportError(ctx, ctx.Err())
	case <-timer.C:
		return nil
	}
}
