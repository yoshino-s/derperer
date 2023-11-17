package fofa

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"errors"

	"go.uber.org/zap"
)

const (
	URL = "https://fofa.info/api/v1/search/next?email=%s&key=%s&qbase64=%s&fields=%s&next=%s&size=%d"
)

type FofaRequest struct {
	Query  string
	Fields string
	Next   string
	Size   int
	Full   string
}

// FofaResponse contains the fofa response
type FofaResponse struct {
	Error   bool       `json:"error"`
	ErrMsg  string     `json:"errmsg"`
	Mode    string     `json:"mode"`
	Next    string     `json:"next"`
	Query   string     `json:"query"`
	Results [][]string `json:"results"`
	Size    int        `json:"size"`
}

type FofaResult struct {
	IP             string `json:"ip"`
	Port           string `json:"port"`
	Host           string `json:"host"`
	Protocol       string `json:"protocol"`
	Country        string `json:"country"`
	Region         string `json:"region"`
	City           string `json:"city"`
	ASOrganization string `json:"as_organization"`
}

type Fofa struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

func (fofa *Fofa) getFields() (string, error) {
	var res FofaResult
	var m map[string]string
	b, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", err
	}

	fields := []string{}
	for k := range m {
		fields = append(fields, k)
	}
	return strings.Join(fields, ","), nil
}

func (fofa *Fofa) Query(query string, batch int, limit int) (<-chan FofaResult, <-chan struct{}, error) {
	if fofa.Email == "" || fofa.Key == "" {
		return nil, nil, errors.New("empty fofa keys")
	}

	fields, err := fofa.getFields()
	if err != nil {
		return nil, nil, err
	}

	results := make(chan FofaResult)
	finish := make(chan struct{})

	go func() {
		defer close(results)

		var numberOfResults int
		var next = ""
		for {
			fofaRequest := &FofaRequest{
				Query:  query,
				Fields: fields,
				Size:   batch,
				Next:   next,
			}
			fofaResponse, err := fofa.query(URL, fofaRequest)
			if err != nil {
				zap.L().Error("failed to query fofa", zap.Error(err))
				if strings.Contains(err.Error(), "45012") { // too fast
					time.Sleep(10 * time.Second)
					continue
				} else {
					break
				}
			}
			for _, result := range fofaResponse.Results {
				m := map[string]string{}
				for i, field := range strings.Split(fields, ",") {
					m[field] = result[i]
				}
				b, err := json.Marshal(m)
				if err != nil {
					zap.L().Error("failed to marshal fofa result", zap.Error(err))
					continue
				}
				var fofaResult FofaResult
				if err := json.Unmarshal(b, &fofaResult); err != nil {
					zap.L().Error("failed to unmarshal fofa result", zap.Error(err))
					continue
				}
				results <- fofaResult
			}

			size := fofaResponse.Size
			if size == 0 || (limit >= 0 && numberOfResults > limit) || len(fofaResponse.Results) == 0 || numberOfResults > size {
				break
			}
			numberOfResults += len(fofaResponse.Results)
			next = fofaResponse.Next

		}
		finish <- struct{}{}
	}()

	return results, finish, nil
}

func (fofa *Fofa) query(URL string, fofaRequest *FofaRequest) (*FofaResponse, error) {
	base64Query := base64.StdEncoding.EncodeToString([]byte(fofaRequest.Query))
	fofaURL := fmt.Sprintf(URL, fofa.Email, fofa.Key, base64Query, fofaRequest.Fields, url.QueryEscape(fofaRequest.Next), fofaRequest.Size)
	response, err := http.Get(fofaURL)
	if err != nil {
		return nil, err
	}

	fofaResponse := &FofaResponse{}

	if err := json.NewDecoder(response.Body).Decode(fofaResponse); err != nil {
		return nil, err
	}
	if fofaResponse.Error {
		return nil, fmt.Errorf(fofaResponse.ErrMsg)
	}
	return fofaResponse, nil
}
