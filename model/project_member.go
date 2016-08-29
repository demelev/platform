// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"
	"io"
	"strings"
)

const (
	PROJECT_ROLE_ADMIN          = "admin"
	PROJECT_NOTIFY_DEFAULT      = "default"
	PROJECT_NOTIFY_ALL          = "all"
	PROJECT_NOTIFY_MENTION      = "mention"
	PROJECT_NOTIFY_NONE         = "none"
	PROJECT_MARK_UNREAD_ALL     = "all"
	PROJECT_MARK_UNREAD_MENTION = "mention"
)

type ProjectMember struct {
	ProjectId string `json:"channel_id"`
	UserId    string `json:"user_id"`
	Roles     string `json:"roles"`
	//CreateAt     int64     `json:"create_at"`
	//UpdateAt     int64     `json:"update_at"`
	DeleteAt     int64     `json:"delete_at"`
	LastViewedAt int64     `json:"last_viewed_at"`
	MsgCount     int64     `json:"msg_count"`
	MentionCount int64     `json:"mention_count"`
	NotifyProps  StringMap `json:"notify_props"`
	LastUpdateAt int64     `json:"last_update_at"`
}

func (o *ProjectMember) ToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func ProjectMemberFromJson(data io.Reader) *ProjectMember {
	decoder := json.NewDecoder(data)
	var o ProjectMember
	err := decoder.Decode(&o)
	if err == nil {
		return &o
	} else {
		return nil
	}
}

func (o *ProjectMember) IsValid() *AppError {

	if len(o.ProjectId) != 26 {
		return NewLocAppError("ProjectMember.IsValid", "model.channel_member.is_valid.channel_id.app_error", nil, "")
	}

	if len(o.UserId) != 26 {
		return NewLocAppError("ProjectMember.IsValid", "model.channel_member.is_valid.user_id.app_error", nil, "")
	}

	for _, role := range strings.Split(o.Roles, " ") {
		if !(role == "" || role == PROJECT_ROLE_ADMIN) {
			return NewLocAppError("ProjectMember.IsValid", "model.channel_member.is_valid.role.app_error", nil, "role="+role)
		}
	}

	notifyLevel := o.NotifyProps["desktop"]
	if len(notifyLevel) > 20 || !IsProjectNotifyLevelValid(notifyLevel) {
		return NewLocAppError("ProjectMember.IsValid", "model.channel_member.is_valid.notify_level.app_error",
			nil, "notify_level="+notifyLevel)
	}

	markUnreadLevel := o.NotifyProps["mark_unread"]
	if len(markUnreadLevel) > 20 || !IsProjectMarkUnreadLevelValid(markUnreadLevel) {
		return NewLocAppError("ProjectMember.IsValid", "model.channel_member.is_valid.unread_level.app_error",
			nil, "mark_unread_level="+markUnreadLevel)
	}

	return nil
}

func (o *ProjectMember) PreSave() {
	o.LastUpdateAt = GetMillis()
}

func (o *ProjectMember) PreUpdate() {
	o.LastUpdateAt = GetMillis()
}

func IsProjectNotifyLevelValid(notifyLevel string) bool {
	return notifyLevel == PROJECT_NOTIFY_DEFAULT ||
		notifyLevel == PROJECT_NOTIFY_ALL ||
		notifyLevel == PROJECT_NOTIFY_MENTION ||
		notifyLevel == PROJECT_NOTIFY_NONE
}

func IsProjectMarkUnreadLevelValid(markUnreadLevel string) bool {
	return markUnreadLevel == PROJECT_MARK_UNREAD_ALL || markUnreadLevel == PROJECT_MARK_UNREAD_MENTION
}

func GetDefaultProjectNotifyProps() StringMap {
	return StringMap{
		"desktop":     PROJECT_NOTIFY_DEFAULT,
		"mark_unread": PROJECT_MARK_UNREAD_ALL,
	}
}
