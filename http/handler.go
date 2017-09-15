package http

import (
	"io"
	"net/http"

	"strings"

	"fmt"

	"github.com/ninech/reception/common"
)

// for every frontend requests, it connects to the backend server
// and relays the exact same response from the backend to the frontend
type BackendHandler struct {
	// maps Host (from request header) to destination Host
	HostMapping *common.HostToHostMap
}

// the http.Handler
func (h BackendHandler) ServeHTTP(frontendResponseWriter http.ResponseWriter, frontendRequest *http.Request) {
	backendRequest := cloneHttpRequest(frontendRequest)

	lookupAndSetDestinationUrl(h.HostMapping, backendRequest, frontendRequest)

	fmt.Printf("%v -> %v (%v)\n", frontendRequest.Host, backendRequest.URL.Host, frontendRequest.RequestURI)

	backendResponse, err := createHttpClient().Do(backendRequest)
	if err != nil {
		frontendResponseWriter.WriteHeader(504)
		return
	}
	defer backendResponse.Body.Close()

	copyResponse(&frontendResponseWriter, backendResponse)
}

// configures the destination of the backend request according to the given frontend request
func lookupAndSetDestinationUrl(hostMapping *common.HostToHostMap, backendRequest, frontendRequest *http.Request) {
	frontendHost := strings.Split(frontendRequest.Host, ":")[0]

	destinationHost, ok := lookupDestinationHost(hostMapping, frontendHost)

	destinationUrl := backendRequest.URL
	destinationUrl.Scheme = "http"

	if ok {
		destinationUrl.Host = destinationHost
	} else {
		destinationUrl.Host = "localhost:8080"
	}
}

// lookup the destination host in the given HostToHostMap
func lookupDestinationHost(hostMapping *common.HostToHostMap, frontendHost string) (destinationHost string, ok bool) {
	if hostMapping != nil {
		hostMapping.RLock()
		destinationHost, ok = hostMapping.M[frontendHost]
		hostMapping.RUnlock()
	}
	return
}

// writes an exact copy of a received http.Response to a http.ResponseWriter
func copyResponse(frontendResponseWriter *http.ResponseWriter, backendResponse *http.Response) {
	// Copy Headers
	headers := (*frontendResponseWriter).Header()
	copyHeaders(&headers, &backendResponse.Header)
	// Send the correct status code
	(*frontendResponseWriter).WriteHeader((*backendResponse).StatusCode)
	// Copy Body
	io.Copy(*frontendResponseWriter, backendResponse.Body)
}

// initializes a new http client, that does not follow redirects
func createHttpClient() *http.Client {
	httpClient := &http.Client{
		CheckRedirect: checkRedirectFunc,
	}
	return httpClient
}

// clones http requests, so that they can be used with http.Client.Do
func cloneHttpRequest(fromRequest *http.Request) (toRequest *http.Request) {
	toRequest = &http.Request{}
	*toRequest = *fromRequest
	toRequest.RequestURI = ""
	return
}

// Copies headers from one http.Header to another
func copyHeaders(toHeader *http.Header, fromHeader *http.Header) {
	for key, values := range *fromHeader {
		for _, value := range values {
			toHeader.Add(key, value)
		}
	}
}

// always returns the first response that is received
// the client that connects to the frontend will handle the redirects
func checkRedirectFunc(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}
