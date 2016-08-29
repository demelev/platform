// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"
	"io"
)

type ProjectsChannels struct {
	ChannelsId   map[string][]string `json:"channels_id"`
	LastUpdateAt int64
}

func (o *ProjectsChannels) Etag() string {
	//var mt int64 = 0
	//if o.ChannelsId != nil {
	//mt = o.Member.LastUpdateAt
	//}

	return Etag(o.LastUpdateAt)
}

func (o *ProjectsChannels) ToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func ProjectsChannelsFromJson(data io.Reader) *ProjectsChannels {
	decoder := json.NewDecoder(data)
	var o ProjectsChannels
	err := decoder.Decode(&o)
	if err == nil {
		return &o
	} else {
		return nil
	}
}
