// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"
	"io"
)

type ProjectData struct {
	Project *Project       `json:"project"`
	Member  *ProjectMember `json:"member"`
}

func (o *ProjectData) Etag() string {
	var mt int64 = 0
	if o.Member != nil {
		mt = o.Member.LastUpdateAt
	}

	return Etag(o.Project.Id, o.Project.UpdateAt, o.Project.LastPostAt, mt)
}

func (o *ProjectData) ToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func ProjectDataFromJson(data io.Reader) *ProjectData {
	decoder := json.NewDecoder(data)
	var o ProjectData
	err := decoder.Decode(&o)
	if err == nil {
		return &o
	} else {
		return nil
	}
}
