package begetapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/miekg/dns"
)

const ErrTemplate = `{"status":"%s","answer":{"status":"%s","errors":[{"error_code":"%s","error_text":%s}]}}`

// Simplified API-mock, accepting json POST's
type BegetApiMock struct {
	login  string
	passwd string
	server *http.Server

	dnsServer  *dns.Server
	txtRecords map[string]Records
	sync.RWMutex
}

func NewBegetApiMock(login string, passwd string) *BegetApiMock {
	return &BegetApiMock{
		login:      login,
		passwd:     passwd,
		txtRecords: make(map[string]Records),
	}
}

func (b *BegetApiMock) Run(addr string) error {
	if b.server != nil {
		return errors.New("server is running")
	}

	mux := http.NewServeMux()
	mux.Handle(
		"/api/dns/changeRecords",
		b.authMiddleware(
			baseParamsCheckMiddleware(
				changeRecordsParamsCheckMiddleware(
					http.HandlerFunc(b.DnsChangeRecords),
				),
			),
		),
	)
	mux.Handle(
		"/api/dns/getData",
		b.authMiddleware(
			baseParamsCheckMiddleware(
				http.HandlerFunc(b.DnsGetData),
			),
		),
	)

	b.server = &http.Server{Addr: addr, Handler: mux}

	return b.server.ListenAndServe()
}

func (b *BegetApiMock) RunDns(port string) {

	b.dnsServer = &dns.Server{
		Addr:    ":" + port,
		Net:     "udp",
		Handler: dns.HandlerFunc(b.handleDNSRequest),
	}

	b.dnsServer.ListenAndServe()
}

func (b *BegetApiMock) Stop(ctx context.Context) error {
	return b.server.Shutdown(ctx)
}

func (b *BegetApiMock) StopDns(_ context.Context) error {
	return b.dnsServer.Shutdown()
}

// API handlers

func (b *BegetApiMock) DnsChangeRecords(w http.ResponseWriter, req *http.Request) {
	fmt.Println("\n\nDnsChangeRecords")

	var v ChangeRecordsRequest
	err := json.Unmarshal([]byte(req.FormValue("input_data")), &v)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(getJsonInvalidError()))
		return
	}

	b.Lock()
	b.txtRecords[v.FQDN] = v.Records
	b.txtRecords[untrimTrimmedFqdn(v.FQDN)] = v.Records // for tests
	b.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","answer":{"status":"success","result":true}}`))
}

// The real API gives back results only if the domain is created in beget's panel
func (b *BegetApiMock) DnsGetData(w http.ResponseWriter, req *http.Request) {
	fmt.Println("\n\nDnsGetData")
	fmt.Println(req.FormValue("input_data"))

	var v GetDataRequest
	err := json.Unmarshal([]byte(req.FormValue("input_data")), &v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(getJsonInvalidError()))
	}

	resp := struct {
		Status string `json:"status"`
		Answer struct {
			Status string        `json:"status"`
			Result GetDataResult `json:"result"`
		} `json:"answer"`
	}{
		Status: "succsess",
	}

	b.Lock()
	resp.Answer.Result.Records = b.txtRecords[v.FQDN]
	b.Unlock()

	if resp.Answer.Result.Records == nil {
		resp.Answer.Result.Records = make(Records, 0)
	}
	resp.Answer.Status = "success"

	response, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(501)
		w.Write([]byte("unexpected mock error: unable to marshal"))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// structs

type GetDataRequest struct {
	FQDN string `json:"fqdn"`
}

// To persist every record and every possible field in it
type Records map[string][]map[string]interface{}

type GetDataResult struct {
	// along with other fields
	FQDN    string  `json:"fqdn"`
	Records Records `json:"records"`
}

type ChangeRecordsRequest struct {
	FQDN    string  `json:"fqdn"`
	Records Records `json:"records"`
}

// helpers

func (b *BegetApiMock) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("middleware")
		r.ParseForm()
		if b.login != r.Form.Get("login") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if b.passwd != r.Form.Get("passwd") {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func baseParamsCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("base middleware")

		r.ParseMultipartForm(1024)
		fmt.Printf("got: %s", r.Form.Encode())

		if r.Form.Has("input_format") && !oneOf[string](r.Form.Get("input_format"), []string{"plain", "json"}) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("<html><body>unhandled error: input_format is no plain|json</body></html>"))
			return
		}
		if r.Form.Has("output_format") && !oneOf[string](r.Form.Get("output_format"), []string{"plain", "json"}) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("<html><body>unhandled error: output_format is no plain|json</body></html>"))
			return
		}

		if r.Form.Get("input_format") == "json" {
			if r.Form.Has("input_data") && !json.Valid([]byte(r.Form.Get("input_data"))) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(getJsonInvalidError()))
				return
			}

			if !r.Form.Has("input_data") {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(getJsonErrorIncorrectInputData()))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func changeRecordsParamsCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("getData middleware")
		r.ParseForm()
		if r.Form.Has("input_format") && !oneOf[string](r.Form.Get("input_format"), []string{"plain", "json"}) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("<html><body>unhandled error</body></html>"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func oneOf[T comparable](value T, values []T) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}
	return false
}

func getJsonError(status, ansStatus, ansErrCode, ansErrText string) string {
	return fmt.Sprintf(ErrTemplate, status, ansStatus, ansErrCode, ansErrText)
}

func getJsonErrorIncorrectInputData() string {
	return getJsonError("success", "error", "INVALID_DATA", "\"Incorrect input\ndata\"")
}

// dns/getData on unkown domain name
func getJsonErrorFailedToGetDnsRecords() string {
	return getJsonError("success", "error", "METHOD_FAILED", "\"Failed to get DNS\nrecords\"")
}

// dns/changeRecords on unkown domain name
func getJsonErrorChangeUnknownDnsRecords() string {
	return getJsonError("success", "error", "METHOD_FAILED", "{\"type\":\"NOT_FOUND_ERROR\",\"message\":null}")
}

func getJsonInvalidError() string {
	return "Cannot parse the JSON input params"
}

func untrimTrimmedFqdn(fqdn string) string {
	return fqdn + "."
}
