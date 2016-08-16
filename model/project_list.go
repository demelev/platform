// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"
	"io"
)

type ProjectList struct {
	Projects []*Project                `json:"projects"`
	Members  map[string]*ProjectMember `json:"members"`
}

func (o *ProjectList) ToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func (o *ProjectList) Etag() string {

	id := "0"
	var t int64 = 0
	var delta int64 = 0

	for _, v := range o.Projects {
		if v.LastPostAt > t {
			t = v.LastPostAt
			id = v.Id
		}

		if v.UpdateAt > t {
			t = v.UpdateAt
			id = v.Id
		}

		member := o.Members[v.Id]

		if member != nil {
			max := v.LastPostAt
			if v.UpdateAt > max {
				max = v.UpdateAt
			}

			delta += max - member.LastViewedAt

			if member.LastViewedAt > t {
				t = member.LastViewedAt
				id = v.Id
			}

			if member.LastUpdateAt > t {
				t = member.LastUpdateAt
				id = v.Id
			}

		}
	}

	return Etag(id, t, delta, len(o.Projects))
}

func ProjectListFromJson(data io.Reader) *ProjectList {
	decoder := json.NewDecoder(data)
	var o ProjectList
	err := decoder.Decode(&o)
	if err == nil {
		return &o
	} else {
		return nil
	}
}
