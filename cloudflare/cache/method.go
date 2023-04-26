package cache

import (
	"errors"
	"net/http"
	"syscall/js"

	"github.com/syumai/workers/internal/jshttp"
	"github.com/syumai/workers/internal/jsutil"
)

// toJSResponse converts *http.Response to JS Response
func toJSResponse(res *http.Response) js.Value {
	status := res.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	respInit := jsutil.NewObject()
	respInit.Set("status", status)
	respInit.Set("statusText", http.StatusText(status))
	respInit.Set("headers", jshttp.ToJSHeader(res.Header))

	readableStream := jsutil.ConvertReaderToReadableStream(res.Body)

	return jsutil.ResponseClass.New(readableStream, respInit)
}

// Put attempts to add a response to the cache, using the given request as the key.
// Returns an error for the following conditions
// - the request passed is a method other than GET.
// - the response passed has a status of 206 Partial Content.
// - Cache-Control instructs not to cache or if the response is too large.
// docs: https://developers.cloudflare.com/workers/runtime-apis/cache/#put
func (c *Cache) Put(req *http.Request, res *http.Response) error {
	_, err := jsutil.AwaitPromise(c.instance.Call("put", jshttp.ToJSRequest(req), toJSResponse(res)))
	if err != nil {
		return err
	}
	return nil
}

// ErrCacheNotFount is returned when there is no matching cache.
var ErrCacheNotFount = errors.New("cache not found")

// MatchOptions represents the options of the Match method.
type MatchOptions struct {
	// IgnoreMethod - Consider the request method a GET regardless of its actual value.
	IgnoreMethod bool
}

// toJS converts MatchOptions to JS object.
func (opts *MatchOptions) toJS() js.Value {
	if opts == nil {
		return js.Undefined()
	}
	obj := jsutil.NewObject()
	obj.Set("ignoreMethod", opts.IgnoreMethod)
	return obj
}

// Match returns the response object keyed to that request.
// docs: https://developers.cloudflare.com/workers/runtime-apis/cache/#match
func (c *Cache) Match(req *http.Request, opts *MatchOptions) (*http.Response, error) {
	res, err := jsutil.AwaitPromise(c.instance.Call("match", jshttp.ToJSRequest(req), opts.toJS()))
	if err != nil {
		return nil, err
	}
	if res.IsUndefined() {
		return nil, ErrCacheNotFount
	}
	return jshttp.ToResponse(res)
}

// DeleteOptions represents the options of the Delete method.
type DeleteOptions struct {
	// IgnoreMethod - Consider the request method a GET regardless of its actual value.
	IgnoreMethod bool
}

// toJS converts DeleteOptions to JS object.
func (opts *DeleteOptions) toJS() js.Value {
	if opts == nil {
		return js.Undefined()
	}
	obj := jsutil.NewObject()
	obj.Set("ignoreMethod", opts.IgnoreMethod)
	return obj
}

// Delete removes the Response object from the cache.
// This method only purges content of the cache in the data center that the Worker was invoked.
// Returns ErrCacheNotFount if the response was not cached.
func (c *Cache) Delete(req *http.Request, opts *DeleteOptions) error {
	res, err := jsutil.AwaitPromise(c.instance.Call("delete", jshttp.ToJSRequest(req), opts.toJS()))
	if err != nil {
		return err
	}
	if !res.Bool() {
		return ErrCacheNotFount
	}
	return nil
}
