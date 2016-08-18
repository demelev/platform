// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"
	"io"
)

type ProjectChannel struct {
	ChannelId string `json:"channel_id"`
	ProjectId string `json:"project_id"`
	Type      string `json:"type"`
}

func (o *ProjectChannel) ToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func ProjectChannelFromJson(data io.Reader) *ProjectChannel {
	decoder := json.NewDecoder(data)
	var o ProjectChannel
	err := decoder.Decode(&o)
	if err == nil {
		return &o
	} else {
		return nil
	}
}

func (o *ProjectChannel) IsValid() *AppError {

	if len(o.ChannelId) != 26 {
		return NewLocAppError("ProjectChannel.IsValid", "model.project_channel.is_valid.channel_id.app_error", nil, "")
	}

	return nil
}

func (o *ProjectChannel) PreSave() {
}

func (o *ProjectChannel) PreUpdate() {
}
