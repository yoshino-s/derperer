package derperer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/derperer/db"
	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/kataras/iris/v12"
	"go.uber.org/zap"
)

const FINGERPRINT = `"<h1>DERP</h1>"`

type Derperer struct {
	DerpererConfig
	*tester
	app *iris.Application
	db  *db.DB
	ctx context.Context
}

type DerpererConfig struct {
	Address        string
	UpdateInterval time.Duration
	FetchInterval  time.Duration
	FofaClient     fofa.Fofa
	LatencyLimit   time.Duration
	ProbeTimeout   time.Duration
	FetchBatch     int
	TestBatch      int
	DatabaseUri    string
}

func NewDerperer(config DerpererConfig) (*Derperer, error) {
	app := iris.New()
	ctx := context.Background()
	t, err := newTester(
		ctx,
		zap.L().Sugar().Infof,
		config.LatencyLimit,
		config.ProbeTimeout,
	)
	if err != nil {
		return nil, err
	}
	db, err := db.New(ctx, config.DatabaseUri)
	if err != nil {
		return nil, err
	}
	derperer := &Derperer{
		DerpererConfig: config,
		tester:         t,
		app:            app,
		db:             db,
		ctx:            ctx,
	}

	app.Get("/derp.json", derperer.getDerpMap)
	app.Get("/webhook", derperer.webhook)

	return derperer, nil
}

func (d *Derperer) FetchFofaData() {
	zap.L().Info("fetching fofa")
	res, finish, err := d.FofaClient.Query(FINGERPRINT, d.FetchBatch, -1)
	if err != nil {
		zap.L().Error("failed to query fofa", zap.Error(err))
	}
	buf := make([]fofa.FofaResult, 0, d.TestBatch)
	func() {
		for {
			select {
			case r := <-res:
				buf = append(buf, r)
				if len(buf) == d.TestBatch {
					d.UpdateDERPMap(buf)
					buf = make([]fofa.FofaResult, 0, d.TestBatch)
				}
			case <-finish:
				return
			}
		}
	}()
}

func (d *Derperer) UpdateDERPMap(rawResult []fofa.FofaResult) {
	derpMap, err := Convert(rawResult)
	if err != nil {
		zap.L().Error("failed to convert", zap.Error(err))
		return
	}
	newDerpMap, _ := d.Test(derpMap)

	for _, region := range newDerpMap.Regions {
		err := d.db.InsertDERPRegion(region)
		if err != nil {
			zap.L().Fatal("failed to insert region", zap.Error(err))
		}
	}
}

func (d *Derperer) Start() error {
	go func() {
		for {
			time.Sleep(d.UpdateInterval)
			derpMap, err := d.db.GetDERPMap()
			if err != nil {
				zap.L().Error("failed to get derp map", zap.Error(err))
				continue
			}
			_, banned := d.Test(derpMap)
			for _, bannedId := range banned {
				d.db.BanRegion(bannedId)
			}
		}
	}()

	go func() {
		for {
			d.FetchFofaData()
			time.Sleep(d.FetchInterval)
		}
	}()

	return d.app.Listen(d.Address)
}

func (d *Derperer) getDerpMap(ctx iris.Context) {
	derpMap, err := d.db.GetDERPMap()
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString(err.Error())
		return
	}
	ctx.JSON(derpMap)
}

func webhookResponse(url string, token string, message string) error {
	body, err := json.Marshal(map[string]string{
		"token":         token,
		"response_type": "in_channel",
		"text":          message,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	_, err = client.Do(req)
	return err

}

func (d *Derperer) webhook(ctx iris.Context) {
	text := ctx.URLParam("text")
	token := ctx.URLParam("token")
	responseUrl := ctx.URLParam("response_url")
	var message string
	regionID, err := strconv.Atoi(text)
	if err != nil {
		message = "region id is invalid"
	} else {
		cnt := 0
		cnt, err := d.db.BanRegion(regionID)
		if err != nil {
			message = "failed to delete region"
		}
		message = fmt.Sprintf("delete %d region(s) for %d", cnt, regionID)
	}
	err = webhookResponse(responseUrl, token, message)
	if err != nil {
		zap.L().Error("failed to send response", zap.Error(err))
	}
	ctx.StatusCode(iris.StatusOK)
	ctx.Text(message)
}
