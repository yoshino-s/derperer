package fofa

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"errors"

	"go.uber.org/zap"
)

const (
	URL    = "https://fofa.info/api/v1/search/all?email=%s&key=%s&qbase64=%s&fields=%s&page=%d&size=%d"
	Fields = "ip,port,host,country,region,as_organization"
)

type Fofa struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

func (fofa *Fofa) Query(query string, batch int, limit int) (<-chan []FofaResult, <-chan struct{}, error) {
	if fofa.Email == "" || fofa.Key == "" {
		return nil, nil, errors.New("empty fofa keys")
	}

	results := make(chan []FofaResult)
	finish := make(chan struct{})

	go func() {
		defer close(results)

		var numberOfResults int
		page := 1
		for {
			fofaRequest := &FofaRequest{
				Query:  query,
				Fields: Fields,
				Size:   batch,
				Page:   page,
			}
			fofaResponse, err := fofa.query(URL, fofaRequest)
			if err != nil {
				zap.L().Error("failed to query fofa", zap.Error(err))
				if strings.Contains(err.Error(), "45012") { // too fast
					time.Sleep(5 * time.Second)
					continue
				} else {
					break
				}
			}
			res := make([]FofaResult, 0, len(fofaResponse.Results))
			for _, result := range fofaResponse.Results {
				res = append(res, FofaResult{
					IP:             result[0],
					Port:           result[1],
					Host:           result[2],
					Country:        result[3],
					Region:         result[4],
					ASOrganization: result[5],
				})
			}
			results <- res
			size := fofaResponse.Size
			if size == 0 || (limit >= 0 && numberOfResults > limit) || len(fofaResponse.Results) == 0 || numberOfResults > size {
				break
			}
			numberOfResults += len(fofaResponse.Results)
			page++

		}
		finish <- struct{}{}
	}()

	return results, finish, nil
}

func (fofa *Fofa) query(URL string, fofaRequest *FofaRequest) (*FofaResponse, error) {
	base64Query := base64.StdEncoding.EncodeToString([]byte(fofaRequest.Query))
	fofaURL := fmt.Sprintf(URL, fofa.Email, fofa.Key, base64Query, Fields, fofaRequest.Page, fofaRequest.Size)
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
	Page   int
	Size   int
	Full   string
}

// FofaResponse contains the fofa response
type FofaResponse struct {
	Error   bool       `json:"error"`
	ErrMsg  string     `json:"errmsg"`
	Mode    string     `json:"mode"`
	Page    int        `json:"page"`
	Query   string     `json:"query"`
	Results [][]string `json:"results"`
	Size    int        `json:"size"`
}

type FofaResult struct {
	IP             string `json:"ip"`
	Port           string `json:"port"`
	Host           string `json:"host"`
	Country        string `json:"country"`
	Region         string `json:"region"`
	ASOrganization string `json:"as_organization"`
}
