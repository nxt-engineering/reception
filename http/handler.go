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
	Config *common.Config
}

// the http.Handler
func (handler BackendHandler) ServeHTTP(frontendResponseWriter http.ResponseWriter, frontendRequest *http.Request) {
	backendRequest := cloneHttpRequest(frontendRequest)

	hostMapping := handler.Config.Projects.AllUrls()

	ok := handler.lookupAndSetDestinationUrl(hostMapping, backendRequest, frontendRequest)
	if !ok {
		frontendResponseWriter.WriteHeader(http.StatusServiceUnavailable)
		fmt.Printf("No backend: %v (%v)\n", frontendRequest.Host, frontendRequest.RequestURI)
		return
	}

	fmt.Printf("%v -> %v (%v)\n", frontendRequest.Host, backendRequest.URL.Host, frontendRequest.RequestURI)

	backendResponse, err := createHttpClient().Do(backendRequest)
	if err != nil {
		frontendResponseWriter.WriteHeader(http.StatusBadGateway)
		return
	}
	defer backendResponse.Body.Close()

	copyResponse(&frontendResponseWriter, backendResponse)
}

// configures the destination of the backend request according to the given frontend request
func (handler BackendHandler) lookupAndSetDestinationUrl(hostMapping map[string]string, backendRequest, frontendRequest *http.Request) (ok bool) {
	frontendHostWithTLD := strings.Split(frontendRequest.Host, ":")[0]
	frontendHost := removeTLD(frontendHostWithTLD, handler.Config.TLD)

	destinationHost, ok := hostMapping[frontendHost]
	if !ok {
		return
	}

	destinationUrl := backendRequest.URL
	destinationUrl.Scheme = "http"

	destinationUrl.Host = destinationHost
	return
}

// removes the TLD from the end of a given hostname:
// given TLD="docker", then: "a.b.docker" -> "a.b", "a.b.docker." -> "a.b"
// given TLD="docker.", then: "a.b.docker" -> "a.b", "a.b.docker." -> "a.b"
func removeTLD(withTLD, TLD string) string {
	cut := len(TLD)

	if "." != lastChar(TLD) {
		cut += 1
	}

	if "." == lastChar(withTLD) {
		cut += 1
	}

	return withTLD[:len(withTLD)-cut]
}

// returns the last character of a string
func lastChar(s string) string {
	return s[len(s)-1:]
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
