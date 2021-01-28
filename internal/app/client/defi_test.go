package client

import (
	"fmt"
	"server-sugar-app/config"
	"testing"
)

func TestGetPledge(t *testing.T) {
	config.Server.DefiPledgeURL = "https://dai.docauthor.com/v1/internal/collateral"
	pledges, err := GetPledge()
	if err != nil {
		t.Fatalf(err.Error())
	}
	for _, plegde := range pledges {
		fmt.Printf("%+v\n", plegde)
	}
}
