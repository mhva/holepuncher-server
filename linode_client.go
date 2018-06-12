package main

import (
	"strconv"

	"net/http"

	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

const linodeAPIBaseURL = "https://api.linode.com/v4"

type paginatedResult interface {
	pageNumber() int
	pageCount() int
	data() interface{}
}

type pageIterator struct {
	request  *resty.Request
	endpoint string
	page     int
}

type apiResult struct {
	data     interface{}
	err      error
	response *resty.Response
}

func linodePOST(endpoint string, r *resty.Request) apiResult {
	return linodeSimpleExec("POST", endpoint, r)
}

func linodeDELETE(endpoint string, r *resty.Request) apiResult {
	return linodeSimpleExec("DELETE", endpoint, r)
}

func linodePUT(endpoint string, r *resty.Request) apiResult {
	return linodeSimpleExec("PUT", endpoint, r)
}

func linodeHEAD(endpoint string, r *resty.Request) apiResult {
	return linodeSimpleExec("HEAD", endpoint, r)
}

func linodeGET(endpoint string, r *resty.Request) apiResult {
	return linodeSimpleExec("GET", endpoint, r)
}

func linodePaginatedGET(endpoint string, r *resty.Request, t paginatedResult) pageIterator {
	iter := pageIterator{
		request:  r,
		endpoint: endpoint,
		page:     1,
	}
	r.Result = t
	return iter
}

func (e *pageIterator) next() (apiResult, bool) {
	if e.page > 1 {
		e.request.SetQueryParam("page", strconv.Itoa(e.page))
	}

	result := linodeSimpleExec("GET", e.endpoint, e.request)
	if result.err != nil {
		return result, false
	}

	response := result.response
	pageInfo, ok := response.Result().(paginatedResult)
	if !ok {
		err := errors.Errorf("Possible API incompatibility: Unable to parse paginated response")
		return apiResult{nil, err, response}, false
	}

	e.page++
	hasMorePages := e.page < pageInfo.pageCount()
	return apiResult{pageInfo.data(), nil, response}, hasMorePages
}

func linodeSimpleExec(method string, endpoint string, r *resty.Request) apiResult {
	var execRequest func(string) (*resty.Response, error)
	switch method {
	case "GET":
		execRequest = r.Get
	case "POST":
		execRequest = r.Post
	case "DELETE":
		execRequest = r.Delete
	case "HEAD":
		execRequest = r.Head
	case "PUT":
		execRequest = r.Put
	case "PATCH":
		execRequest = r.Patch
	default:
		panic("Unknown request method: " + method)
	}

	response, err := execRequest(linodeAPIBaseURL + endpoint)
	if err != nil {
		err = errors.Wrapf(err, "%s request ('%s') failed", method, endpoint)
		return apiResult{nil, err, response}
	}

	if response.StatusCode() > 299 {
		errObject := response.Error()
		errFormat := "API error (%s '%s'): %s"
		if errObject != nil {
			if linodeErr, ok := errObject.(*LinodeError); ok {
				linodeErr.isAuthError = response.StatusCode() == http.StatusUnauthorized
				linodeErr.isPermissionsError = response.StatusCode() == http.StatusForbidden
				err = linodeErr
			} else {
				err = errors.Errorf(errFormat, method, endpoint, errObject)
			}
		} else {
			err = errors.Errorf(errFormat, method, endpoint, "No error object, details missing")
		}
		return apiResult{nil, err, response}
	}

	return apiResult{response.Result(), nil, response}
}
