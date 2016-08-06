// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"
	"io"
	"unicode/utf8"
)

const (
	PROJECT_OPEN    = "O"
	PROJECT_PRIVATE = "P"
)

type Project struct {
	Id            string `json:"id"`
	CreateAt      int64  `json:"create_at"`
	UpdateAt      int64  `json:"update_at"`
	DeleteAt      int64  `json:"delete_at"`
	TeamId        string `json:"team_id"`
	Type          string `json:"type"`
	DisplayName   string `json:"display_name"`
	Name          string `json:"name"`
	Header        string `json:"header"`
	LastPostAt    int64  `json:"last_post_at"`
	ExtraUpdateAt int64  `json:"extra_update_at"`
	CreatorId     string `json:"creator_id"`
}

func (o *Project) ToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func ProjectFromJson(data io.Reader) *Project {
	decoder := json.NewDecoder(data)
	var o Project
	err := decoder.Decode(&o)
	if err == nil {
		return &o
	} else {
		return nil
	}
}

func (o *Project) Etag() string {
	return Etag(o.Id, o.UpdateAt)
}

func (o *Project) ExtraEtag(memberLimit int) string {
	return Etag(o.Id, o.ExtraUpdateAt, memberLimit)
}

func (o *Project) IsValid() *AppError {

	if len(o.Id) != 26 {
		return NewLocAppError("Project.IsValid", "model.project.is_valid.id.app_error", nil, "")
	}

	if o.CreateAt == 0 {
		return NewLocAppError("Project.IsValid", "model.project.is_valid.create_at.app_error", nil, "id="+o.Id)
	}

	if o.UpdateAt == 0 {
		return NewLocAppError("Project.IsValid", "model.project.is_valid.update_at.app_error", nil, "id="+o.Id)
	}

	if utf8.RuneCountInString(o.DisplayName) > 64 {
		return NewLocAppError("project.IsValid", "model.project.is_valid.display_name.app_error", nil, "id="+o.Id)
	}

	if len(o.Name) > 64 {
		return NewLocAppError("project.IsValid", "model.project.is_valid.name.app_error", nil, "id="+o.Id)
	}

	if utf8.RuneCountInString(o.Header) > 1024 {
		return NewLocAppError("Project.IsValid", "model.project.is_valid.header.app_error", nil, "id="+o.Id)
	}

	if len(o.CreatorId) > 26 {
		return NewLocAppError("Project.IsValid", "model.project.is_valid.creator_id.app_error", nil, "")
	}

	return nil
}

func (o *Project) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	o.CreateAt = GetMillis()
	o.UpdateAt = o.CreateAt
	o.ExtraUpdateAt = o.CreateAt
}

func (o *Project) PreUpdate() {
	o.UpdateAt = GetMillis()
}

func (o *Project) ExtraUpdated() {
	o.ExtraUpdateAt = GetMillis()
}

func (o *Project) Sanitize() {
}

func (o *Project) SanitizeForNotLoggedIn() {
}

func ProjectMapToJson(u map[string]*Project) string {
	b, err := json.Marshal(u)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func ProjectMapFromJson(data io.Reader) map[string]*Project {
	decoder := json.NewDecoder(data)
	var projects map[string]*Project
	err := decoder.Decode(&projects)
	if err == nil {
		return projects
	} else {
		return nil
	}
}
