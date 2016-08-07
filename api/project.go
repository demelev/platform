// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api

import (
	//"bytes"
	"fmt"
	//"html/template"
	"net/http"
	//"net/url"
	"strconv"
	"strings"
	//"time"

	l4g "github.com/alecthomas/log4go"
	"github.com/gorilla/mux"

	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/utils"
)

func InitProject() {
	l4g.Debug(utils.T("api.project.init.debug"))

	BaseRoutes.Projects.Handle("/create", ApiAppHandler(createProject)).Methods("POST")
	BaseRoutes.Projects.Handle("/all", ApiAppHandler(getAllProjects)).Methods("GET")
	BaseRoutes.Projects.Handle("/all_project_listings", ApiUserRequired(GetAllProjectListings)).Methods("GET")
	BaseRoutes.Projects.Handle("/find_project_by_name", ApiAppHandler(findProjectByName)).Methods("POST")

	BaseRoutes.NeedProject.Handle("/members/{id:[A-Za-z0-9]+}", ApiUserRequired(getProjectMembers)).Methods("GET")
	BaseRoutes.NeedProject.Handle("/me", ApiUserRequired(getMyProject)).Methods("GET")
	BaseRoutes.NeedProject.Handle("/update", ApiUserRequired(updateProject)).Methods("POST")

	BaseRoutes.NeedProject.Handle("/invite_members", ApiUserRequired(inviteMembersToProject)).Methods("POST")

	BaseRoutes.NeedProject.Handle("/add_user_to_project", ApiUserRequired(addUserToProject)).Methods("POST")
	BaseRoutes.NeedProject.Handle("/remove_user_from_project", ApiUserRequired(removeUserFromProject)).Methods("POST")

	// These should be moved to the global admin console
	//BaseRoutes.NeedProject.Handle("/import_project", ApiUserRequired(importProject)).Methods("POST")
	BaseRoutes.Projects.Handle("/add_user_to_project_from_invite", ApiUserRequired(addUserToProjectFromInvite)).Methods("POST")
}

func createProject(c *Context, w http.ResponseWriter, r *http.Request) {
	project := model.ProjectFromJson(r.Body)

	if project == nil {
		c.SetInvalidParam("createProject", "project")
		return
	}

	var user *model.User
	if len(c.Session.UserId) > 0 {
		uchan := Srv.Store.User().Get(c.Session.UserId)

		if result := <-uchan; result.Err != nil {
			c.Err = result.Err
			return
		} else {
			user = result.Data.(*model.User)
		}
	}

	rproject := CreateProject(c, project)
	if c.Err != nil {
		return
	}

	if user != nil {
		err := JoinUserToProject(project, user)
		if err != nil {
			c.Err = err
			return
		}
	}

	w.Write([]byte(rproject.ToJson()))
}

func CreateProject(c *Context, project *model.Project) *model.Project {

	if project == nil {
		c.SetInvalidParam("createProject", "project")
		return nil
	}

	//if !isProjectCreationAllowed(c, project.Email) {
	//return nil
	//}

	if result := <-Srv.Store.Project().Save(project); result.Err != nil {
		c.Err = result.Err
		return nil
	} else {
		rproject := result.Data.(*model.Project)

		if _, err := CreateDefaultChannels(c, rproject.Id); err != nil {
			c.Err = err
			return nil
		}

		return rproject
	}
}

func JoinUserToProjectById(projectId string, user *model.User) *model.AppError {
	if result := <-Srv.Store.Project().Get(projectId); result.Err != nil {
		return result.Err
	} else {
		return JoinUserToProject(result.Data.(*model.Project), user)
	}
}

func JoinUserToProject(project *model.Project, user *model.User) *model.AppError {

	tm := &model.ProjectMember{ProjectId: project.Id, UserId: user.Id}

	//channelRole := ""
	//if project.Email == user.Email {
	//tm.Roles = model.ROLE_TEAM_ADMIN
	//channelRole = model.CHANNEL_ROLE_ADMIN
	//}

	if etmr := <-Srv.Store.Project().GetMember(project.Id, user.Id); etmr.Err == nil {
		// Membership alredy exists.  Check if deleted and and update, otherwise do nothing
		rtm := etmr.Data.(model.ProjectMember)

		// Do nothing if already added
		if rtm.DeleteAt == 0 {
			return nil
		}

		if tmr := <-Srv.Store.Project().UpdateMember(tm); tmr.Err != nil {
			return tmr.Err
		}
	} else {
		// Membership appears to be missing.  Lets try to add.
		if tmr := <-Srv.Store.Project().SaveMember(tm); tmr.Err != nil {
			return tmr.Err
		}
	}

	if uua := <-Srv.Store.User().UpdateUpdateAt(user.Id); uua.Err != nil {
		return uua.Err
	}

	// Soft error if there is an issue joining the default channels
	//if err := JoinDefaultChannels(project.Id, user, channelRole); err != nil {
	//l4g.Error(utils.T("api.user.create_user.joining.error"), user.Id, project.Id, err)
	//}

	RemoveAllSessionsForUserId(user.Id)
	InvalidateCacheForUser(user.Id)

	// This message goes to every channel, so the channelId is irrelevant
	go Publish(model.NewWebSocketEvent("", "", user.Id, model.WEBSOCKET_EVENT_NEW_USER))

	return nil
}

func LeaveProject(project *model.Project, user *model.User) *model.AppError {

	var projectMember model.ProjectMember

	if result := <-Srv.Store.Project().GetMember(project.Id, user.Id); result.Err != nil {
		return model.NewLocAppError("RemoveUserFromProject", "api.project.remove_user_from_project.missing.app_error", nil, result.Err.Error())
	} else {
		projectMember = result.Data.(model.ProjectMember)
	}

	var channelMembers *model.ChannelList

	if result := <-Srv.Store.Channel().GetChannels(project.Id, user.Id); result.Err != nil {
		if result.Err.Id == "store.sql_channel.get_channels.not_found.app_error" {
			channelMembers = &model.ChannelList{make([]*model.Channel, 0), make(map[string]*model.ChannelMember)}
		} else {
			return result.Err
		}

	} else {
		channelMembers = result.Data.(*model.ChannelList)
	}

	for _, channel := range channelMembers.Channels {
		if channel.Type != model.CHANNEL_DIRECT {
			if result := <-Srv.Store.Channel().RemoveMember(channel.Id, user.Id); result.Err != nil {
				return result.Err
			}
		}
	}

	if result := <-Srv.Store.Project().UpdateMember(&projectMember); result.Err != nil {
		return result.Err
	}

	if uua := <-Srv.Store.User().UpdateUpdateAt(user.Id); uua.Err != nil {
		return uua.Err
	}

	RemoveAllSessionsForUserId(user.Id)
	InvalidateCacheForUser(user.Id)

	go Publish(model.NewWebSocketEvent(project.Id, "", user.Id, model.WEBSOCKET_EVENT_LEAVE_TEAM))

	return nil
}

func isProjectCreationAllowed(c *Context, email string) bool {

	email = strings.ToLower(email)

	if !c.IsSystemAdmin() && !utils.Cfg.ProjectSettings.EnableProjectCreation {
		c.Err = model.NewLocAppError("isProjectCreationAllowed", "api.project.is_project_creation_allowed.disabled.app_error", nil, "")
		return false
	}

	if result := <-Srv.Store.User().GetByEmail(email); result.Err == nil {
		user := result.Data.(*model.User)
		if len(user.AuthService) > 0 && len(*user.AuthData) > 0 {
			return true
		}
	}

	return true
}

func GetAllProjectListings(c *Context, w http.ResponseWriter, r *http.Request) {
	if result := <-Srv.Store.Project().GetAllProjectListing(); result.Err != nil {
		c.Err = result.Err
		return
	} else {
		projects := result.Data.([]*model.Project)
		m := make(map[string]*model.Project)
		for _, v := range projects {
			m[v.Id] = v
			if !c.IsSystemAdmin() {
				m[v.Id].Sanitize()
			}
		}

		w.Write([]byte(model.ProjectMapToJson(m)))
	}
}

func getAllProjects(c *Context, w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	teamId := params["team_id"]

	if result := <-Srv.Store.Project().GetAll(teamId); result.Err != nil {
		c.Err = result.Err
		return
	} else {
		projects := result.Data.([]*model.Project)
		m := make(map[string]*model.Project)
		for _, v := range projects {
			m[v.Id] = v
			if !c.IsSystemAdmin() {
				m[v.Id].SanitizeForNotLoggedIn()
			}
		}

		w.Write([]byte(model.ProjectMapToJson(m)))
	}
}

func inviteMembersToProject(c *Context, w http.ResponseWriter, r *http.Request) {
	invites := model.InvitesFromJson(r.Body)

	params := mux.Vars(r)
	projectId := params["project_id"]

	if len(invites.Invites) == 0 {
		c.Err = model.NewLocAppError("inviteMembers", "api.project.invite_members.no_one.app_error", nil, "")
		c.Err.StatusCode = http.StatusBadRequest
		return
	}

	if utils.IsLicensed {
		if *utils.Cfg.ProjectSettings.RestrictProjectInvite == model.PERMISSIONS_SYSTEM_ADMIN && !c.IsSystemAdmin() {
			c.Err = model.NewLocAppError("inviteMembers", "api.project.invite_members.restricted_system_admin.app_error", nil, "")
			return
		}

		if *utils.Cfg.ProjectSettings.RestrictProjectInvite == model.PERMISSIONS_PROJECT_ADMIN && !c.IsProjectAdmin() {
			c.Err = model.NewLocAppError("inviteMembers", "api.project.invite_members.restricted_project_admin.app_error", nil, "")
			return
		}
	}

	tchan := Srv.Store.Project().Get(projectId)
	uchan := Srv.Store.User().Get(c.Session.UserId)

	var project *model.Project
	if result := <-tchan; result.Err != nil {
		c.Err = result.Err
		return
	} else {
		project = result.Data.(*model.Project)
	}

	var user *model.User
	if result := <-uchan; result.Err != nil {
		c.Err = result.Err
		return
	} else {
		user = result.Data.(*model.User)
	}

	emailList := make([]string, len(invites.Invites))
	for _, invite := range invites.Invites {
		emailList = append(emailList, invite["email"])
	}

	InviteMembersToProject(c, project, user, emailList)

	w.Write([]byte(invites.ToJson()))
}

func addUserToProject(c *Context, w http.ResponseWriter, r *http.Request) {
	params := model.MapFromJson(r.Body)
	userId := params["user_id"]

	url_params := mux.Vars(r)
	projectId := url_params["project_id"]

	if len(projectId) != 26 {
		c.SetInvalidParam("addUserToProject", "project_id")
		return
	}

	if len(userId) != 26 {
		c.SetInvalidParam("addUserToProject", "user_id")
		return
	}

	tchan := Srv.Store.Project().Get(projectId)
	uchan := Srv.Store.User().Get(userId)

	var project *model.Project
	if result := <-tchan; result.Err != nil {
		c.Err = result.Err
		return
	} else {
		project = result.Data.(*model.Project)
	}

	var user *model.User
	if result := <-uchan; result.Err != nil {
		c.Err = result.Err
		return
	} else {
		user = result.Data.(*model.User)
	}

	if !c.IsProjectAdmin() {
		c.Err = model.NewLocAppError("addUserToProject", "api.project.update_project.permissions.app_error", nil, "userId="+c.Session.UserId)
		c.Err.StatusCode = http.StatusForbidden
		return
	}

	err := JoinUserToProject(project, user)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.MapToJson(params)))
}

func removeUserFromProject(c *Context, w http.ResponseWriter, r *http.Request) {
	params := model.MapFromJson(r.Body)
	userId := params["user_id"]

	url_params := mux.Vars(r)
	projectId := url_params["project_id"]

	if len(projectId) != 26 {
		c.SetInvalidParam("removeUserFromProject", "project_id")
		return
	}

	if len(userId) != 26 {
		c.SetInvalidParam("removeUserFromProject", "user_id")
		return
	}

	tchan := Srv.Store.Project().Get(projectId)
	uchan := Srv.Store.User().Get(userId)

	var project *model.Project
	if result := <-tchan; result.Err != nil {
		c.Err = result.Err
		return
	} else {
		project = result.Data.(*model.Project)
	}

	var user *model.User
	if result := <-uchan; result.Err != nil {
		c.Err = result.Err
		return
	} else {
		user = result.Data.(*model.User)
	}

	if c.Session.UserId != user.Id {
		if !c.IsProjectAdmin() {
			c.Err = model.NewLocAppError("removeUserFromProject", "api.project.update_project.permissions.app_error", nil, "userId="+c.Session.UserId)
			c.Err.StatusCode = http.StatusForbidden
			return
		}
	}

	err := LeaveProject(project, user)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.MapToJson(params)))
}

func addUserToProjectFromInvite(c *Context, w http.ResponseWriter, r *http.Request) {

	params := model.MapFromJson(r.Body)
	hash := params["hash"]
	data := params["data"]
	inviteId := params["invite_id"]

	projectId := ""
	var project *model.Project

	if len(hash) > 0 {
		props := model.MapFromJson(strings.NewReader(data))

		if !model.ComparePassword(hash, fmt.Sprintf("%v:%v", data, utils.Cfg.EmailSettings.InviteSalt)) {
			c.Err = model.NewLocAppError("addUserToProjectFromInvite", "api.user.create_user.signup_link_invalid.app_error", nil, "")
			return
		}

		t, err := strconv.ParseInt(props["time"], 10, 64)
		if err != nil || model.GetMillis()-t > 1000*60*60*48 { // 48 hours
			c.Err = model.NewLocAppError("addUserToProjectFromInvite", "api.user.create_user.signup_link_expired.app_error", nil, "")
			return
		}

		projectId = props["id"]

		// try to load the project to make sure it exists
		if result := <-Srv.Store.Project().Get(projectId); result.Err != nil {
			c.Err = result.Err
			return
		} else {
			project = result.Data.(*model.Project)
		}
	}

	if len(inviteId) > 0 {
		if result := <-Srv.Store.Project().GetByInviteId(inviteId); result.Err != nil {
			c.Err = result.Err
			return
		} else {
			project = result.Data.(*model.Project)
			projectId = project.Id
		}
	}

	if len(projectId) == 0 {
		c.Err = model.NewLocAppError("addUserToProjectFromInvite", "api.user.create_user.signup_link_invalid.app_error", nil, "")
		return
	}

	uchan := Srv.Store.User().Get(c.Session.UserId)

	var user *model.User
	if result := <-uchan; result.Err != nil {
		c.Err = result.Err
		return
	} else {
		user = result.Data.(*model.User)
	}

	/*
	 *    tm := c.Session.GetProjectByProjectId(projectId)
	 *
	 *    if tm == nil {
	 *        err := JoinUserToProject(project, user)
	 *        if err != nil {
	 *            c.Err = err
	 *            return
	 *        }
	 *    }
	 */

	err := JoinUserToProject(project, user)
	if err != nil {
		c.Err = err
		return
	}

	project.Sanitize()

	w.Write([]byte(project.ToJson()))
}

func FindProjectByName(teamId string, name string) bool {
	if result := <-Srv.Store.Project().GetByName(teamId, name); result.Err != nil {
		return false
	} else {
		return true
	}
}

func findProjectByName(c *Context, w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	teamId := params["team_id"]
	//TODO: add checking

	m := model.MapFromJson(r.Body)
	name := strings.ToLower(strings.TrimSpace(m["name"]))

	found := FindProjectByName(teamId, name)

	if found {
		w.Write([]byte("true"))
	} else {
		w.Write([]byte("false"))
	}
}

func InviteMembersToProject(c *Context, project *model.Project, user *model.User, invites []string) {
	for _, invite := range invites {
		if len(invite) > 0 {

			//sender := user.GetDisplayName()
			//TODO: implement it

			/*
			 *            senderRole := ""
			 *            if c.IsProjectAdmin() {
			 *                senderRole = c.T("api.project.invite_members.admin")
			 *            } else {
			 *                senderRole = c.T("api.project.invite_members.member")
			 *            }
			 *
			 *            subjectPage := utils.NewHTMLTemplate("invite_subject", c.Locale)
			 *            subjectPage.Props["Subject"] = c.T("api.templates.invite_subject",
			 *                map[string]interface{}{"SenderName": sender, "ProjectDisplayName": project.DisplayName, "SiteName": utils.ClientCfg["SiteName"]})
			 *
			 *            bodyPage := utils.NewHTMLTemplate("invite_body", c.Locale)
			 *            bodyPage.Props["SiteURL"] = c.GetSiteURL()
			 *            bodyPage.Props["Title"] = c.T("api.templates.invite_body.title")
			 *            bodyPage.Html["Info"] = template.HTML(c.T("api.templates.invite_body.info",
			 *                map[string]interface{}{"SenderStatus": senderRole, "SenderName": sender, "ProjectDisplayName": project.DisplayName}))
			 *            bodyPage.Props["Button"] = c.T("api.templates.invite_body.button")
			 *            bodyPage.Html["ExtraInfo"] = template.HTML(c.T("api.templates.invite_body.extra_info",
			 *                map[string]interface{}{"ProjectDisplayName": project.DisplayName, "ProjectURL": c.GetProjectURL()}))
			 *
			 *            props := make(map[string]string)
			 *            props["email"] = invite
			 *            props["id"] = project.Id
			 *            props["display_name"] = project.DisplayName
			 *            props["name"] = project.Name
			 *            props["time"] = fmt.Sprintf("%v", model.GetMillis())
			 *            data := model.MapToJson(props)
			 *            hash := model.HashPassword(fmt.Sprintf("%v:%v", data, utils.Cfg.EmailSettings.InviteSalt))
			 *            bodyPage.Props["Link"] = fmt.Sprintf("%s/signup_user_complete/?d=%s&h=%s", c.GetSiteURL(), url.QueryEscape(data), url.QueryEscape(hash))
			 *
			 *            if !utils.Cfg.EmailSettings.SendEmailNotifications {
			 *                l4g.Info(utils.T("api.project.invite_members.sending.info"), invite, bodyPage.Props["Link"])
			 *            }
			 *
			 *            if err := utils.SendMail(invite, subjectPage.Render(), bodyPage.Render()); err != nil {
			 *                l4g.Error(utils.T("api.project.invite_members.send.error"), err)
			 *            }
			 */
		}
	}
}

func updateProject(c *Context, w http.ResponseWriter, r *http.Request) {

	project := model.ProjectFromJson(r.Body)
	params := mux.Vars(r)
	projectId := params["project_id"]

	if project == nil {
		c.SetInvalidParam("updateProject", "project")
		return
	}

	project.Id = projectId

	if !c.IsProjectAdmin() {
		c.Err = model.NewLocAppError("updateProject", "api.project.update_project.permissions.app_error", nil, "userId="+c.Session.UserId)
		c.Err.StatusCode = http.StatusForbidden
		return
	}

	var oldProject *model.Project
	if result := <-Srv.Store.Project().Get(project.Id); result.Err != nil {
		c.Err = result.Err
		return
	} else {
		oldProject = result.Data.(*model.Project)
	}

	oldProject.DisplayName = project.DisplayName
	//oldProject.InviteId = project.InviteId
	//oldProject.AllowOpenInvite = project.AllowOpenInvite
	//oldProject.CompanyName = project.CompanyName
	//oldProject.AllowedDomains = project.AllowedDomains
	//oldProject.Type = project.Type

	if result := <-Srv.Store.Project().Update(oldProject); result.Err != nil {
		c.Err = result.Err
		return
	}

	oldProject.Sanitize()

	w.Write([]byte(oldProject.ToJson()))
}

func PermanentDeleteProject(c *Context, project *model.Project) *model.AppError {
	l4g.Warn(utils.T("api.project.permanent_delete_project.attempting.warn"), project.Name, project.Id)
	c.Path = "/projects/permanent_delete"
	c.LogAuditWithUserId("", fmt.Sprintf("attempt projectId=%v", project.Id))

	project.DeleteAt = model.GetMillis()
	if result := <-Srv.Store.Project().Update(project); result.Err != nil {
		return result.Err
	}

	if result := <-Srv.Store.Project().PermanentDeleteByProject(project.Id); result.Err != nil {
		return result.Err
	}

	if result := <-Srv.Store.Project().RemoveAllMembersByProject(project.Id); result.Err != nil {
		return result.Err
	}

	if result := <-Srv.Store.Project().PermanentDelete(project.Id); result.Err != nil {
		return result.Err
	}

	l4g.Warn(utils.T("api.project.permanent_delete_project.deleted.warn"), project.Name, project.Id)
	c.LogAuditWithUserId("", fmt.Sprintf("success projectId=%v", project.Id))

	return nil
}

func getMyProject(c *Context, w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	projectId := params["project_id"]

	if len(projectId) == 0 {
		return
	}

	if result := <-Srv.Store.Project().Get(projectId); result.Err != nil {
		c.Err = result.Err
		return
	} else if HandleEtag(result.Data.(*model.Project).Etag(), w, r) {
		return
	} else {
		w.Header().Set(model.HEADER_ETAG_SERVER, result.Data.(*model.Project).Etag())
		w.Write([]byte(result.Data.(*model.Project).ToJson()))
		return
	}
}

//func importProject(c *Context, w http.ResponseWriter, r *http.Request) {
//if !c.HasPermissionsToProject(c.ProjectId, "import") || !c.IsProjectAdmin() {
//c.Err = model.NewLocAppError("importProject", "api.project.import_project.admin.app_error", nil, "userId="+c.Session.UserId)
//c.Err.StatusCode = http.StatusForbidden
//return
//}

//if err := r.ParseMultipartForm(10000000); err != nil {
//c.Err = model.NewLocAppError("importProject", "api.project.import_project.parse.app_error", nil, err.Error())
//return
//}

//importFromArray, ok := r.MultipartForm.Value["importFrom"]
//importFrom := importFromArray[0]

//fileSizeStr, ok := r.MultipartForm.Value["filesize"]
//if !ok {
//c.Err = model.NewLocAppError("importProject", "api.project.import_project.unavailable.app_error", nil, "")
//c.Err.StatusCode = http.StatusBadRequest
//return
//}

//fileSize, err := strconv.ParseInt(fileSizeStr[0], 10, 64)
//if err != nil {
//c.Err = model.NewLocAppError("importProject", "api.project.import_project.integer.app_error", nil, "")
//c.Err.StatusCode = http.StatusBadRequest
//return
//}

//fileInfoArray, ok := r.MultipartForm.File["file"]
//if !ok {
//c.Err = model.NewLocAppError("importProject", "api.project.import_project.no_file.app_error", nil, "")
//c.Err.StatusCode = http.StatusBadRequest
//return
//}

//if len(fileInfoArray) <= 0 {
//c.Err = model.NewLocAppError("importProject", "api.project.import_project.array.app_error", nil, "")
//c.Err.StatusCode = http.StatusBadRequest
//return
//}

//fileInfo := fileInfoArray[0]

//fileData, err := fileInfo.Open()
//defer fileData.Close()
//if err != nil {
//c.Err = model.NewLocAppError("importProject", "api.project.import_project.open.app_error", nil, err.Error())
//c.Err.StatusCode = http.StatusBadRequest
//return
//}

//var log *bytes.Buffer
//switch importFrom {
//case "slack":
//var err *model.AppError
//if err, log = SlackImport(fileData, fileSize, c.ProjectId); err != nil {
//c.Err = err
//c.Err.StatusCode = http.StatusBadRequest
//}
//}

//w.Header().Set("Content-Disposition", "attachment; filename=MattermostImportLog.txt")
//w.Header().Set("Content-Type", "application/octet-stream")
//http.ServeContent(w, r, "MattermostImportLog.txt", time.Now(), bytes.NewReader(log.Bytes()))
//}

func getProjectMembers(c *Context, w http.ResponseWriter, r *http.Request) {
	//params := mux.Vars(r)
	//id := params["id"]

	//TODO: implement it
	/*
	 *    if c.Session.GetProjectByProjectId(id) == nil {
	 *        if !c.HasSystemAdminPermissions("getProjectMembers") {
	 *            return
	 *        }
	 *    }
	 *
	 *    if result := <-Srv.Store.Project().GetProjectMembers(id); result.Err != nil {
	 *        c.Err = result.Err
	 *        return
	 *    } else {
	 *        members := result.Data.([]*model.ProjectMember)
	 *        w.Write([]byte(model.ProjectMembersToJson(members)))
	 *        return
	 *    }
	 */
}
