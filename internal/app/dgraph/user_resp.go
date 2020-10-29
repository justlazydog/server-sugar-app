package dgraph

// UserResp represent user struct returned by dgraph
type UserResp struct {
	Name     string     `json:"n,omitempty"`
	Nickname string     `json:"c,omitempty"`
	Avatar   string     `json:"a,omitempty"`
	Links    []UserResp `json:"l,omitempty"`

	Point map[string]float64 `json:"v,omitempty"`
}

// Walk recursively do fn to every user in user tree.
func (u UserResp) Walk(us []UserResp, depth int, fn func(u UserResp, depth int)) {
	for _, u := range us {
		fn(u, depth)
		if len(u.Links) > 0 {
			nextDepth := depth + 1
			u.Walk(u.Links, nextDepth, fn)
		}
	}
}
