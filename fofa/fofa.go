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
	URL    = "https://fofa.info/api/v1/search/next?email=%s&key=%s&qbase64=%s&fields=%s&next=%s&size=%d"
	Fields = "ip,port,host,protocol,country,region,as_organization"
)

type Fofa struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

func (fofa *Fofa) Query(query string, batch int, limit int) (<-chan FofaResult, <-chan struct{}, error) {
	if fofa.Email == "" || fofa.Key == "" {
		return nil, nil, errors.New("empty fofa keys")
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
				Fields: Fields,
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
				results <- FofaResult{
					IP:             result[0],
					Port:           result[1],
					Host:           result[2],
					Protocol:       result[3],
					Country:        result[4],
					Region:         result[5],
					ASOrganization: result[6],
				}
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
	fofaURL := fmt.Sprintf(URL, fofa.Email, fofa.Key, base64Query, Fields, url.QueryEscape(fofaRequest.Next), fofaRequest.Size)
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
	ASOrganization string `json:"as_organization"`
}
