// Package apiv2 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package apiv2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/oapi-codegen/runtime"
)

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// GetSiteProductivity request
	GetSiteProductivity(ctx context.Context, siteID string, params *GetSiteProductivityParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetSiteTaskDistribution request
	GetSiteTaskDistribution(ctx context.Context, siteID string, params *GetSiteTaskDistributionParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetSiteUtilization request
	GetSiteUtilization(ctx context.Context, siteID string, params *GetSiteUtilizationParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetWorkerTaskHistory request
	GetWorkerTaskHistory(ctx context.Context, workerID string, params *GetWorkerTaskHistoryParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetWorkerProductivity request
	GetWorkerProductivity(ctx context.Context, workerID string, params *GetWorkerProductivityParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetWorkerUtilization request
	GetWorkerUtilization(ctx context.Context, workerID string, params *GetWorkerUtilizationParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetWorkerCurrentTasks request
	GetWorkerCurrentTasks(ctx context.Context, workflowID string, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetTaskBlockageStatus request
	GetTaskBlockageStatus(ctx context.Context, workflowID string, taskID string, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) GetSiteProductivity(ctx context.Context, siteID string, params *GetSiteProductivityParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetSiteProductivityRequest(c.Server, siteID, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetSiteTaskDistribution(ctx context.Context, siteID string, params *GetSiteTaskDistributionParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetSiteTaskDistributionRequest(c.Server, siteID, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetSiteUtilization(ctx context.Context, siteID string, params *GetSiteUtilizationParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetSiteUtilizationRequest(c.Server, siteID, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetWorkerTaskHistory(ctx context.Context, workerID string, params *GetWorkerTaskHistoryParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetWorkerTaskHistoryRequest(c.Server, workerID, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetWorkerProductivity(ctx context.Context, workerID string, params *GetWorkerProductivityParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetWorkerProductivityRequest(c.Server, workerID, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetWorkerUtilization(ctx context.Context, workerID string, params *GetWorkerUtilizationParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetWorkerUtilizationRequest(c.Server, workerID, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetWorkerCurrentTasks(ctx context.Context, workflowID string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetWorkerCurrentTasksRequest(c.Server, workflowID)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetTaskBlockageStatus(ctx context.Context, workflowID string, taskID string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetTaskBlockageStatusRequest(c.Server, workflowID, taskID)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewGetSiteProductivityRequest generates requests for GetSiteProductivity
func NewGetSiteProductivityRequest(server string, siteID string, params *GetSiteProductivityParams) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "siteID", runtime.ParamLocationPath, siteID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/sites/%s/productivity", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "start", runtime.ParamLocationQuery, params.Start); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "end", runtime.ParamLocationQuery, params.End); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetSiteTaskDistributionRequest generates requests for GetSiteTaskDistribution
func NewGetSiteTaskDistributionRequest(server string, siteID string, params *GetSiteTaskDistributionParams) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "siteID", runtime.ParamLocationPath, siteID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/sites/%s/tasks", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.Start != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "start", runtime.ParamLocationQuery, *params.Start); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		if params.End != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "end", runtime.ParamLocationQuery, *params.End); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetSiteUtilizationRequest generates requests for GetSiteUtilization
func NewGetSiteUtilizationRequest(server string, siteID string, params *GetSiteUtilizationParams) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "siteID", runtime.ParamLocationPath, siteID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/sites/%s/utilization", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.Date != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "date", runtime.ParamLocationQuery, *params.Date); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetWorkerTaskHistoryRequest generates requests for GetWorkerTaskHistory
func NewGetWorkerTaskHistoryRequest(server string, workerID string, params *GetWorkerTaskHistoryParams) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "workerID", runtime.ParamLocationPath, workerID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/workers/%s/history", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.Start != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "start", runtime.ParamLocationQuery, *params.Start); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		if params.End != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "end", runtime.ParamLocationQuery, *params.End); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetWorkerProductivityRequest generates requests for GetWorkerProductivity
func NewGetWorkerProductivityRequest(server string, workerID string, params *GetWorkerProductivityParams) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "workerID", runtime.ParamLocationPath, workerID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/workers/%s/productivity", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.Start != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "start", runtime.ParamLocationQuery, *params.Start); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		if params.End != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "end", runtime.ParamLocationQuery, *params.End); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetWorkerUtilizationRequest generates requests for GetWorkerUtilization
func NewGetWorkerUtilizationRequest(server string, workerID string, params *GetWorkerUtilizationParams) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "workerID", runtime.ParamLocationPath, workerID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/workers/%s/utilization", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.Date != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "date", runtime.ParamLocationQuery, *params.Date); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetWorkerCurrentTasksRequest generates requests for GetWorkerCurrentTasks
func NewGetWorkerCurrentTasksRequest(server string, workflowID string) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "workflowID", runtime.ParamLocationPath, workflowID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/workflow/%s/tasks", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetTaskBlockageStatusRequest generates requests for GetTaskBlockageStatus
func NewGetTaskBlockageStatusRequest(server string, workflowID string, taskID string) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "workflowID", runtime.ParamLocationPath, workflowID)
	if err != nil {
		return nil, err
	}

	var pathParam1 string

	pathParam1, err = runtime.StyleParamWithLocation("simple", false, "taskID", runtime.ParamLocationPath, taskID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/workflow/%s/tasks/%s/blockage", pathParam0, pathParam1)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// GetSiteProductivityWithResponse request
	GetSiteProductivityWithResponse(ctx context.Context, siteID string, params *GetSiteProductivityParams, reqEditors ...RequestEditorFn) (*GetSiteProductivityResponse, error)

	// GetSiteTaskDistributionWithResponse request
	GetSiteTaskDistributionWithResponse(ctx context.Context, siteID string, params *GetSiteTaskDistributionParams, reqEditors ...RequestEditorFn) (*GetSiteTaskDistributionResponse, error)

	// GetSiteUtilizationWithResponse request
	GetSiteUtilizationWithResponse(ctx context.Context, siteID string, params *GetSiteUtilizationParams, reqEditors ...RequestEditorFn) (*GetSiteUtilizationResponse, error)

	// GetWorkerTaskHistoryWithResponse request
	GetWorkerTaskHistoryWithResponse(ctx context.Context, workerID string, params *GetWorkerTaskHistoryParams, reqEditors ...RequestEditorFn) (*GetWorkerTaskHistoryResponse, error)

	// GetWorkerProductivityWithResponse request
	GetWorkerProductivityWithResponse(ctx context.Context, workerID string, params *GetWorkerProductivityParams, reqEditors ...RequestEditorFn) (*GetWorkerProductivityResponse, error)

	// GetWorkerUtilizationWithResponse request
	GetWorkerUtilizationWithResponse(ctx context.Context, workerID string, params *GetWorkerUtilizationParams, reqEditors ...RequestEditorFn) (*GetWorkerUtilizationResponse, error)

	// GetWorkerCurrentTasksWithResponse request
	GetWorkerCurrentTasksWithResponse(ctx context.Context, workflowID string, reqEditors ...RequestEditorFn) (*GetWorkerCurrentTasksResponse, error)

	// GetTaskBlockageStatusWithResponse request
	GetTaskBlockageStatusWithResponse(ctx context.Context, workflowID string, taskID string, reqEditors ...RequestEditorFn) (*GetTaskBlockageStatusResponse, error)
}

type GetSiteProductivityResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *SiteProductivityMetrics
	JSON400      *Error
	JSON500      *Error
}

// Status returns HTTPResponse.Status
func (r GetSiteProductivityResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetSiteProductivityResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetSiteTaskDistributionResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *SiteTaskDistribution
	JSON400      *Error
	JSON500      *Error
}

// Status returns HTTPResponse.Status
func (r GetSiteTaskDistributionResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetSiteTaskDistributionResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetSiteUtilizationResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *SiteUtilizationMetrics
	JSON400      *Error
	JSON500      *Error
}

// Status returns HTTPResponse.Status
func (r GetSiteUtilizationResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetSiteUtilizationResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetWorkerTaskHistoryResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *WorkerTaskHistory
	JSON400      *Error
	JSON500      *Error
}

// Status returns HTTPResponse.Status
func (r GetWorkerTaskHistoryResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetWorkerTaskHistoryResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetWorkerProductivityResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *WorkerProductivityMetrics
	JSON400      *Error
	JSON500      *Error
}

// Status returns HTTPResponse.Status
func (r GetWorkerProductivityResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetWorkerProductivityResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetWorkerUtilizationResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *WorkerUtilizationMetrics
	JSON400      *Error
	JSON500      *Error
}

// Status returns HTTPResponse.Status
func (r GetWorkerUtilizationResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetWorkerUtilizationResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetWorkerCurrentTasksResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *WorkerTaskStatus
	JSON400      *Error
	JSON500      *Error
}

// Status returns HTTPResponse.Status
func (r GetWorkerCurrentTasksResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetWorkerCurrentTasksResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetTaskBlockageStatusResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *TaskBlockageInfo
	JSON400      *Error
	JSON500      *Error
}

// Status returns HTTPResponse.Status
func (r GetTaskBlockageStatusResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetTaskBlockageStatusResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// GetSiteProductivityWithResponse request returning *GetSiteProductivityResponse
func (c *ClientWithResponses) GetSiteProductivityWithResponse(ctx context.Context, siteID string, params *GetSiteProductivityParams, reqEditors ...RequestEditorFn) (*GetSiteProductivityResponse, error) {
	rsp, err := c.GetSiteProductivity(ctx, siteID, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetSiteProductivityResponse(rsp)
}

// GetSiteTaskDistributionWithResponse request returning *GetSiteTaskDistributionResponse
func (c *ClientWithResponses) GetSiteTaskDistributionWithResponse(ctx context.Context, siteID string, params *GetSiteTaskDistributionParams, reqEditors ...RequestEditorFn) (*GetSiteTaskDistributionResponse, error) {
	rsp, err := c.GetSiteTaskDistribution(ctx, siteID, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetSiteTaskDistributionResponse(rsp)
}

// GetSiteUtilizationWithResponse request returning *GetSiteUtilizationResponse
func (c *ClientWithResponses) GetSiteUtilizationWithResponse(ctx context.Context, siteID string, params *GetSiteUtilizationParams, reqEditors ...RequestEditorFn) (*GetSiteUtilizationResponse, error) {
	rsp, err := c.GetSiteUtilization(ctx, siteID, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetSiteUtilizationResponse(rsp)
}

// GetWorkerTaskHistoryWithResponse request returning *GetWorkerTaskHistoryResponse
func (c *ClientWithResponses) GetWorkerTaskHistoryWithResponse(ctx context.Context, workerID string, params *GetWorkerTaskHistoryParams, reqEditors ...RequestEditorFn) (*GetWorkerTaskHistoryResponse, error) {
	rsp, err := c.GetWorkerTaskHistory(ctx, workerID, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetWorkerTaskHistoryResponse(rsp)
}

// GetWorkerProductivityWithResponse request returning *GetWorkerProductivityResponse
func (c *ClientWithResponses) GetWorkerProductivityWithResponse(ctx context.Context, workerID string, params *GetWorkerProductivityParams, reqEditors ...RequestEditorFn) (*GetWorkerProductivityResponse, error) {
	rsp, err := c.GetWorkerProductivity(ctx, workerID, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetWorkerProductivityResponse(rsp)
}

// GetWorkerUtilizationWithResponse request returning *GetWorkerUtilizationResponse
func (c *ClientWithResponses) GetWorkerUtilizationWithResponse(ctx context.Context, workerID string, params *GetWorkerUtilizationParams, reqEditors ...RequestEditorFn) (*GetWorkerUtilizationResponse, error) {
	rsp, err := c.GetWorkerUtilization(ctx, workerID, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetWorkerUtilizationResponse(rsp)
}

// GetWorkerCurrentTasksWithResponse request returning *GetWorkerCurrentTasksResponse
func (c *ClientWithResponses) GetWorkerCurrentTasksWithResponse(ctx context.Context, workflowID string, reqEditors ...RequestEditorFn) (*GetWorkerCurrentTasksResponse, error) {
	rsp, err := c.GetWorkerCurrentTasks(ctx, workflowID, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetWorkerCurrentTasksResponse(rsp)
}

// GetTaskBlockageStatusWithResponse request returning *GetTaskBlockageStatusResponse
func (c *ClientWithResponses) GetTaskBlockageStatusWithResponse(ctx context.Context, workflowID string, taskID string, reqEditors ...RequestEditorFn) (*GetTaskBlockageStatusResponse, error) {
	rsp, err := c.GetTaskBlockageStatus(ctx, workflowID, taskID, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetTaskBlockageStatusResponse(rsp)
}

// ParseGetSiteProductivityResponse parses an HTTP response from a GetSiteProductivityWithResponse call
func ParseGetSiteProductivityResponse(rsp *http.Response) (*GetSiteProductivityResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetSiteProductivityResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest SiteProductivityMetrics
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseGetSiteTaskDistributionResponse parses an HTTP response from a GetSiteTaskDistributionWithResponse call
func ParseGetSiteTaskDistributionResponse(rsp *http.Response) (*GetSiteTaskDistributionResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetSiteTaskDistributionResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest SiteTaskDistribution
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseGetSiteUtilizationResponse parses an HTTP response from a GetSiteUtilizationWithResponse call
func ParseGetSiteUtilizationResponse(rsp *http.Response) (*GetSiteUtilizationResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetSiteUtilizationResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest SiteUtilizationMetrics
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseGetWorkerTaskHistoryResponse parses an HTTP response from a GetWorkerTaskHistoryWithResponse call
func ParseGetWorkerTaskHistoryResponse(rsp *http.Response) (*GetWorkerTaskHistoryResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetWorkerTaskHistoryResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest WorkerTaskHistory
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseGetWorkerProductivityResponse parses an HTTP response from a GetWorkerProductivityWithResponse call
func ParseGetWorkerProductivityResponse(rsp *http.Response) (*GetWorkerProductivityResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetWorkerProductivityResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest WorkerProductivityMetrics
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseGetWorkerUtilizationResponse parses an HTTP response from a GetWorkerUtilizationWithResponse call
func ParseGetWorkerUtilizationResponse(rsp *http.Response) (*GetWorkerUtilizationResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetWorkerUtilizationResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest WorkerUtilizationMetrics
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseGetWorkerCurrentTasksResponse parses an HTTP response from a GetWorkerCurrentTasksWithResponse call
func ParseGetWorkerCurrentTasksResponse(rsp *http.Response) (*GetWorkerCurrentTasksResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetWorkerCurrentTasksResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest WorkerTaskStatus
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseGetTaskBlockageStatusResponse parses an HTTP response from a GetTaskBlockageStatusWithResponse call
func ParseGetTaskBlockageStatusResponse(rsp *http.Response) (*GetTaskBlockageStatusResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetTaskBlockageStatusResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest TaskBlockageInfo
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}
