package dgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// RelationSep used split two user when using string to represent relation.
const (
	RelationSep    = ","
	facetsTotalKey = "total"
)

// ListRelation list all relation with it's point.
func ListRelation(pageSize int) (map[string]float64, error) {
	first := pageSize
	offset := 0
	relationValueMapper := make(map[string]float64)

	for {
		var q = fmt.Sprintf(`
query data() {
	data(func: has(name), first:%d, offset:%d) {
		n:name
	    l:cons @facets(v:%s) {
			n:name
	    }
	}
}`, first, offset, facetsTotalKey)
		offset += first
		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, time.Minute)
		resp, err := Dg.NewReadOnlyTxn().BestEffort().Query(ctx, q)
		if err != nil {
			return nil, err
		}
		fmt.Println("query relation ok")

		type Root struct {
			Users []UserResp `json:"data"`
		}

		var r Root
		err = json.Unmarshal(resp.Json, &r)
		if err != nil {
			return nil, err
		}
		if len(r.Users) == 0 {
			break
		}

		UserResp{}.Walk(r.Users, 0, func(u UserResp, depth int) {
			for i, child := range u.Links {
				k := u.Name + RelationSep + child.Name
				v := u.Point[strconv.Itoa(i)]
				relationValueMapper[k] = v
			}
		})
	}

	return relationValueMapper, nil
}
