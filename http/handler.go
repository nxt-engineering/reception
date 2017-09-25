package http

import (
	"html/template"
	"io"
	"net/http"

	"strings"

	"fmt"

	"github.com/ninech/reception/common"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"

	"github.com/GeertJohan/go.rice"
)

// for every frontend requests, it connects to the backend server
// and relays the exact same response from the backend to the frontend
type BackendHandler struct {
	// maps Host (from request header) to destination Host
	Config *common.Config
}

// the http.Handler
func (handler BackendHandler) ServeHTTP(frontendResponseWriter http.ResponseWriter, frontendRequest *http.Request) {
	if handler.isReceptionRequest(frontendRequest) {
		handler.handleReception(frontendResponseWriter, frontendRequest, false)
		return
	}

	backendRequest := cloneHttpRequest(frontendRequest)
	ok := handler.lookupAndSetDestinationUrl(backendRequest, frontendRequest)
	if !ok {
		fmt.Printf("No backend: %v (%v)\n", frontendRequest.Host, frontendRequest.RequestURI)
		handler.handleReception(frontendResponseWriter, frontendRequest, true)
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

func (handler BackendHandler) isReceptionRequest(frontendRequest *http.Request) bool {
	frontendHostWithTLD := strings.Split(frontendRequest.Host, ":")[0]
	frontendHost := removeTLD(frontendHostWithTLD, handler.Config.TLD)
	return "reception" == frontendHost
}

func (handler BackendHandler) handleReception(
	frontendResponseWriter http.ResponseWriter,
	frontendRequest *http.Request,
	isFallback bool) {

	templateBox, err := rice.FindBox("../resources")
	if err != nil {
		frontendResponseWriter.WriteHeader(http.StatusInternalServerError)
		panic(err)
	}

	templateString, err := templateBox.String("index.html.tmpl")

	tmpl, err := template.New("index.html").Parse(templateString)
	if err != nil {
		frontendResponseWriter.WriteHeader(http.StatusInternalServerError)
		panic(err)
	}

	frontendResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	frontendResponseWriter.WriteHeader(http.StatusOK)

	tld := handler.Config.TLD
	if "." == tld[len(tld)-1:] {
		tld = tld[:len(tld)-1]
	}

	data := struct {
		TLD      string
		Projects map[string]*common.Project
		NotFound bool
	}{
		TLD:      tld,
		Projects: handler.Config.Projects.M,
		NotFound: isFallback,
	}

	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/javascript", js.Minify)
	minifiedResponseWriter := m.ResponseWriter(frontendResponseWriter, frontendRequest)
	defer minifiedResponseWriter.Close()

	handler.Config.Projects.RLock()
	defer handler.Config.Projects.RUnlock()
	err = tmpl.Execute(minifiedResponseWriter, data)
	if err != nil {
		panic(err)
	}
}

// configures the destination of the backend request according to the given frontend request
func (handler BackendHandler) lookupAndSetDestinationUrl(backendRequest, frontendRequest *http.Request) (ok bool) {
	frontendHostWithTLD := strings.Split(frontendRequest.Host, ":")[0]
	frontendHost := removeTLD(frontendHostWithTLD, handler.Config.TLD)

	hostMapping := handler.Config.Projects.AllUrls()
	//fmt.Printf("hostMapping: %v\n", hostMapping)

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
	TLD = normalizeHostname(TLD)
	withTLD = normalizeHostname(withTLD)

	if TLD == withTLD {
		return ""
	}

	return withTLD[:len(withTLD)-len(TLD)-1]
}

func normalizeHostname(hostname string) string {
	lastChar := hostname[len(hostname)-1:]

	if "." != lastChar {
		return hostname + "."
	} else {
		return hostname
	}
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
