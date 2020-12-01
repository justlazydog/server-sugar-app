package dao

import (
	"server-sugar-app/internal/db"
)

type LockedSIE struct {
	UID    string  `json:"uid"`
	Volume float64 `json:"volume"`
}

func GetLockedSIE() ([]LockedSIE, error) {
	results := make([]LockedSIE, 0)
	sql := `
select 
	uid, sum(locked) volume
from
	ctc_orders
where
	state = 'wait' 
	and is_robot <> 1
	and quote = 'ask'
	and market = 'siesusd'
group by uid
`
	rows, err := db.ExchangeMysqlCli.Query(sql)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		result := LockedSIE{}
		err = rows.Scan(&result.UID, &result.Volume)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}
