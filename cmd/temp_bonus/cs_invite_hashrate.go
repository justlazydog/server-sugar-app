package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"server-sugar-app/internal/app/dgraph"
	"server-sugar-app/internal/app/group"
	"strings"
	"time"
)

const (
	rootUser        = "a3b640a1-1de0-4fe8-9b57-b3985256efb2"
	csDGraphRPCAddr = "open.isecret.im:9082"
)

func main() {

	err := dgraph.Open(csDGraphRPCAddr)
	if err != nil {
		log.Fatalf("open cs d-graph failed: %v", err)
	}

	// prepare group relation
	group.GetLatestGroupRela()
	if !group.RelateUpdated {
		log.Fatalf("update relation failed")
	}

	// prepare cs amount
	csAmounts, err := getCsAmounts()
	if err != nil {
		log.Fatalf("get cs amounts failed: %v", err)
	}
	log.Infof("get cs amount done")

	// prepare team amount file
	path := fmt.Sprintf("team_amount_%s.txt", time.Now().Format("20060102150405"))
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("create team amount file failed: %v", err)
	}
	defer f.Close()

	ctx := &calcContext{
		csAmounts:      csAmounts,
		teamAmounts:    make(map[string]float64),
		inviteRelation: make(map[string]string),
	}

	ctx.calcTeamAmount(rootUser)
	log.Infof("calc team amount done.")

	// write team amount file.
	// teamAmounts contains all user.
	for uid, teamAmount := range ctx.teamAmounts {
		csAmount := ctx.csAmounts[uid]
		if csAmount > 0 || teamAmount > 0 {
			// uid, cs数量, 团队数量, 邀请人uid
			_, err := f.WriteString(fmt.Sprintf("%s,%.6f,%.6f,%s\n", uid, csAmount, teamAmount, ctx.inviteRelation[uid]))
			if err != nil {
				log.Fatalf("write file failed: %v", err)
			}
		}
	}

}

type calcContext struct {
	csAmounts      map[string]float64
	teamAmounts    map[string]float64
	inviteRelation map[string]string // child:parent
}

func (ctx *calcContext) calcTeamAmount(uid string) float64 {
	myTeamAmount := ctx.csAmounts[uid]
	children := group.GetDownLineUsers(uid)
	for _, child := range children {
		ctx.inviteRelation[child] = uid
		myTeamAmount += ctx.calcTeamAmount(child)
	}
	ctx.teamAmounts[uid] = myTeamAmount
	return myTeamAmount
}

// 获取cs直接连接数量
func getCsAmounts() (map[string]float64, error) {
	relations, err := dgraph.ListRelation(5000)
	if err != nil {
		return nil, fmt.Errorf("list relation from d-graph failed: %v", err)
	}
	csAmounts := make(map[string]float64)
	for relation, val := range relations {
		tmp := strings.Split(relation, dgraph.RelationSep)
		if len(tmp) != 2 {
			return nil, fmt.Errorf("bad relation %s", relation)
		}
		csAmounts[tmp[0]] += val
		csAmounts[tmp[1]] += val
	}
	return csAmounts, nil
}
