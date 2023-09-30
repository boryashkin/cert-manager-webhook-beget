package beget

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

const TXTKey = "TXT"
const TXTDataKey = "txtdata"

type Credentials struct {
	Login  string
	Passwd string
}

type ApiClient struct {
	apiURL *url.URL
	client *http.Client
}

func NewApiClient(apiURL *url.URL) *ApiClient {
	client := http.Client{}

	q := apiURL.Query()
	q.Add("input_format", "json")
	q.Add("output_format", "json")

	apiURL.RawQuery = q.Encode()

	return &ApiClient{
		apiURL: apiURL,
		client: &client,
	}
}

func (a *ApiClient) GetData(fqdn string, credentials Credentials) (Records, error) {
	u := *a.apiURL
	u.Path += "/api/dns/getData"

	q := u.Query()
	q.Add("login", credentials.Login)
	q.Add("passwd", credentials.Passwd)

	u.RawQuery = q.Encode()

	values := map[string]string{"fqdn": fqdn}

	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal a message: %w", err)
	}

	buff := bytes.NewBuffer([]byte(""))
	mp := multipart.NewWriter(buff)

	err = mp.WriteField("input_data", string(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("failed to write form data: %w", err)
	}

	err = mp.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close form data: %w", err)
	}

	r, err := a.client.Post(u.String(), mp.FormDataContentType(), buff)
	if err != nil {
		return nil, fmt.Errorf("request for getData failed: %w", err)
	}

	if r.StatusCode != 200 {
		return nil, fmt.Errorf("non 200 response: %d", r.StatusCode)
	}

	bdy, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	r.Body.Close()

	var rsp GetDataResponse

	err = json.Unmarshal(bdy, &rsp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	fmt.Println(string(bdy), rsp)

	return rsp.Answer.Result.Records, nil
}

func (a *ApiClient) ChangeRecords(fqdn string, records Records, credentials Credentials) error {
	u := *a.apiURL
	u.Path += "/api/dns/changeRecords"

	q := u.Query()
	q.Add("login", credentials.Login)
	q.Add("passwd", credentials.Passwd)

	u.RawQuery = q.Encode()

	values := struct {
		FQDN    string  `json:"fqdn"`
		Records Records `json:"records"`
	}{
		FQDN:    fqdn,
		Records: records,
	}

	jsonValue, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal a message: %w", err)
	}

	buff := bytes.NewBuffer([]byte(""))
	mp := multipart.NewWriter(buff)

	err = mp.WriteField("input_data", string(jsonValue))
	if err != nil {
		return fmt.Errorf("failed to write form data: %w", err)
	}

	err = mp.Close()
	if err != nil {
		return fmt.Errorf("failed to close form data: %w", err)
	}

	r, err := a.client.Post(u.String(), mp.FormDataContentType(), buff)
	if err != nil {
		return fmt.Errorf("request for getData failed: %w", err)
	}

	bdy, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	r.Body.Close()

	if r.StatusCode != 200 {
		return fmt.Errorf("non 200 response: %d %s", r.StatusCode, bdy)
	}

	var result ChangeRecordsResponse
	err = json.Unmarshal(bdy, &result)
	if err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if !result.Answer.Result {
		return fmt.Errorf("got result status in response: %s, body: %s", result.Answer.Status, bdy)
	}

	return nil
}

type GetDataResponse struct {
	Status string `json:"status"`
	Answer struct {
		Status string        `json:"status"`
		Result GetDataResult `json:"result"`
	} `json:"answer"`
}

type ChangeRecordsResponse struct {
	Status string `json:"status"`
	Answer struct {
		Status string `json:"status"`
		Result bool   `json:"result"`
	} `json:"answer"`
}

func PushTXTRecord(r Records, txtData string) error {
	if _, ok := r[TXTKey]; ok && len(r[TXTKey]) > 0 {
		for i, _ := range r[TXTKey] {
			existingData, isStr := r[TXTKey][i][TXTDataKey].(string)
			if isStr && existingData == txtData {
				return nil
			}
		}

		var newTxtElement map[string]interface{}
		r[TXTKey] = append(r[TXTKey], newTxtElement)

		r[TXTKey][len(r[TXTKey])-1][TXTDataKey] = txtData
	} else {
		if r == nil {
			r = make(Records)
		}
		r[TXTKey] = make([]map[string]interface{}, 1)
		r[TXTKey][0] = make(map[string]interface{}, 1)
		r[TXTKey][0][TXTDataKey] = txtData
	}

	return nil
}

func PopTXTRecordByValue(r Records, txtData string) int {
	deleted := 0
	if _, ok := r[TXTKey]; ok && len(r[TXTKey]) > 0 {
		for i := range r[TXTKey] {
			if r[TXTKey][i] == nil {
				continue
			}

			if _, ok := r[TXTKey][i][TXTDataKey]; !ok {
				continue
			}

			existingData, isStr := r[TXTKey][i][TXTDataKey].(string)
			if isStr && existingData == txtData {
				r[TXTKey] = append(r[TXTKey][:i], r[TXTKey][i+1:]...)

				deleted++
			}
		}
	}

	return deleted
}
