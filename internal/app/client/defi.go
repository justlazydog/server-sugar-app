package client

import (
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"net/http"
	"server-sugar-app/config"
)

type DefiPledge struct {
	UID    string          `json:"-"`
	OpenID string          `json:"open_id"`
	Token  string          `json:"token"`
	Amount decimal.Decimal `json:"amount"`
}

type PledgeResp struct {
	Code int          `json:"code"`
	Data []DefiPledge `json:"data"`
}

func GetPledge() ([]DefiPledge, error) {
	resp, err := http.Get(config.Server.DefiPledgeURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result PledgeResp
	if err := json.Unmarshal(bs, &result); err != nil {
		return nil, fmt.Errorf("unmarshal %s to %T failed: %v", string(bs), result, err)
	}
	return result.Data, nil
}
