// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package store

import (
	//"database/sql"
	"github.com/go-gorp/gorp"
	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/utils"
)

const (
	MISSING_PROJECT_ERROR = "store.sql_project.get_by_name.missing.app_error"
	PROJECT_EXISTS_ERROR  = "store.sql_project.save_project.exists.app_error"
)

type SqlProjectStore struct {
	*SqlStore
}

func NewSqlProjectStore(sqlStore *SqlStore) ProjectStore {
	s := &SqlProjectStore{sqlStore}

	for _, db := range sqlStore.GetAllConns() {
		table := db.AddTableWithName(model.Project{}, "Projects").SetKeys(false, "Id")
		table.ColMap("Id").SetMaxSize(26)
		table.ColMap("TeamId").SetMaxSize(26)
		table.ColMap("Type").SetMaxSize(1)
		table.ColMap("DisplayName").SetMaxSize(64)
		table.ColMap("Name").SetMaxSize(64)
		//table.SetUniqueTogether("Name", "TeamId")
		table.ColMap("Header").SetMaxSize(1024)
		table.ColMap("CreatorId").SetMaxSize(26)

		tablem := db.AddTableWithName(model.ProjectMember{}, "ProjectMembers").SetKeys(false, "ProjectId", "UserId")
		tablem.ColMap("ProjectId").SetMaxSize(26)
		tablem.ColMap("UserId").SetMaxSize(26)
		tablem.ColMap("Roles").SetMaxSize(64)
		tablem.ColMap("NotifyProps").SetMaxSize(2000)
	}

	return s
}

func (s SqlProjectStore) UpgradeSchemaIfNeeded() {
}

func (s SqlProjectStore) CreateIndexesIfNotExists() {
	s.CreateIndexIfNotExists("idx_projects_team_id", "Projects", "TeamId")
	s.CreateIndexIfNotExists("idx_projects_name", "Projects", "Name")

	s.CreateIndexIfNotExists("idx_projectmembers_project_id", "ProjectMembers", "ProjectId")
	s.CreateIndexIfNotExists("idx_projectmembers_user_id", "ProjectMembers", "UserId")
}

func (s SqlProjectStore) Save(project *model.Project) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		var result StoreResult
		//if project.Type == model.project_DIRECT {
		//result.Err = model.NewLocAppError("SqlProjectStore.Save", "store.sql_project.save.direct_project.app_error", nil, "")
		//} else {
		//if transaction, err := s.GetMaster().Begin(); err != nil {
		//result.Err = model.NewLocAppError("SqlProjectStore.Save", "store.sql_project.save.open_transaction.app_error", nil, err.Error())
		//} else {
		//result = s.saveProjectT(transaction, project)
		//if result.Err != nil {
		//transaction.Rollback()
		//} else {
		//if err := transaction.Commit(); err != nil {
		//result.Err = model.NewLocAppError("SqlProjectStore.Save", "store.sql_project.save.commit_transaction.app_error", nil, err.Error())
		//}
		//}
		//}
		//}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) saveProjectT(transaction *gorp.Transaction, project *model.Project) StoreResult {
	result := StoreResult{}

	if len(project.Id) > 0 {
		result.Err = model.NewLocAppError("SqlProjectStore.Save", "store.sql_project.save_project.existing.app_error", nil, "id="+project.Id)
		return result
	}

	project.PreSave()
	if result.Err = project.IsValid(); result.Err != nil {
		return result
	}

	//if project.Type != model.project_DIRECT {
	//if count, err := transaction.SelectInt("SELECT COUNT(0) FROM Projects WHERE TeamId = :TeamId AND DeleteAt = 0 AND (Type = 'O' OR Type = 'P')", map[string]interface{}{"TeamId": project.TeamId}); err != nil {
	//result.Err = model.NewLocAppError("SqlProjectStore.Save", "store.sql_project.save_project.current_count.app_error", nil, "teamId="+project.TeamId+", "+err.Error())
	//return result
	//} else if count > 1000 {
	//result.Err = model.NewLocAppError("SqlProjectStore.Save", "store.sql_project.save_project.limit.app_error", nil, "teamId="+project.TeamId)
	//return result
	//}
	//}

	//if err := transaction.Insert(project); err != nil {
	//if IsUniqueConstraintError(err.Error(), []string{"Name", "projects_name_teamid_key"}) {
	//dupProject := model.Project{}
	//s.GetMaster().SelectOne(&dupProject, "SELECT * FROM Projects WHERE TeamId = :TeamId AND Name = :Name AND DeleteAt > 0", map[string]interface{}{"TeamId": project.TeamId, "Name": project.Name})
	//if dupProject.DeleteAt > 0 {
	//result.Err = model.NewLocAppError("SqlProjectStore.Save", "store.sql_project.save_project.previously.app_error", nil, "id="+project.Id+", "+err.Error())
	//} else {
	//result.Err = model.NewLocAppError("SqlProjectStore.Save", project_EXISTS_ERROR, nil, "id="+project.Id+", "+err.Error())
	//result.Data = &dupProject
	//}
	//} else {
	//result.Err = model.NewLocAppError("SqlProjectStore.Save", "store.sql_project.save_project.save.app_error", nil, "id="+project.Id+", "+err.Error())
	//}
	//} else {
	//result.Data = project
	//}

	return result
}

func (s SqlProjectStore) Update(project *model.Project) StoreChannel {

	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		project.PreUpdate()

		if result.Err = project.IsValid(); result.Err != nil {
			storeProject <- result
			close(storeProject)
			return
		}

		if count, err := s.GetMaster().Update(project); err != nil {
			if IsUniqueConstraintError(err.Error(), []string{"Name", "projects_name_teamid_key"}) {
				dupProject := model.Project{}
				s.GetReplica().SelectOne(&dupProject, "SELECT * FROM Projects WHERE TeamId = :TeamId AND Name= :Name AND DeleteAt > 0", map[string]interface{}{"TeamId": project.TeamId, "Name": project.Name})
				if dupProject.DeleteAt > 0 {
					result.Err = model.NewLocAppError("SqlProjectStore.Update", "store.sql_project.update.previously.app_error", nil, "id="+project.Id+", "+err.Error())
				} else {
					result.Err = model.NewLocAppError("SqlProjectStore.Update", "store.sql_project.update.exists.app_error", nil, "id="+project.Id+", "+err.Error())
				}
			} else {
				result.Err = model.NewLocAppError("SqlProjectStore.Update", "store.sql_project.update.updating.app_error", nil, "id="+project.Id+", "+err.Error())
			}
		} else if count != 1 {
			result.Err = model.NewLocAppError("SqlProjectStore.Update", "store.sql_project.update.app_error", nil, "id="+project.Id)
		} else {
			result.Data = project
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) extraUpdated(project *model.Project) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		project.ExtraUpdated()

		_, err := s.GetMaster().Exec(
			`UPDATE
				Projects
			SET
				ExtraUpdateAt = :Time
			WHERE
				Id = :Id`,
			map[string]interface{}{"Id": project.Id, "Time": project.ExtraUpdateAt})

		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.extraUpdated", "store.sql_project.extra_updated.app_error", nil, "id="+project.Id+", "+err.Error())
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) Get(id string) StoreChannel {
	return s.get(id, false)
}

func (s SqlProjectStore) GetFromMaster(id string) StoreChannel {
	return s.get(id, true)
}

func (s SqlProjectStore) get(id string, master bool) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		var db *gorp.DbMap
		if master {
			db = s.GetMaster()
		} else {
			db = s.GetReplica()
		}

		if obj, err := db.Get(model.Project{}, id); err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.Get", "store.sql_project.get.find.app_error", nil, "id="+id+", "+err.Error())
		} else if obj == nil {
			result.Err = model.NewLocAppError("SqlProjectStore.Get", "store.sql_project.get.existing.app_error", nil, "id="+id)
		} else {
			result.Data = obj.(*model.Project)
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) Delete(projectId string, time int64) StoreChannel {
	return s.SetDeleteAt(projectId, time, time)
}

func (s SqlProjectStore) SetDeleteAt(projectId string, deleteAt int64, updateAt int64) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		_, err := s.GetMaster().Exec("Update Projects SET DeleteAt = :DeleteAt, UpdateAt = :UpdateAt WHERE Id = :ProjectId", map[string]interface{}{"DeleteAt": deleteAt, "UpdateAt": updateAt, "ProjectId": projectId})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.Delete", "store.sql_project.delete.project.app_error", nil, "id="+projectId+", err="+err.Error())
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) PermanentDelete(projectId string) StoreChannel {
	storeProject := make(StoreChannel)
	return storeProject
}

func (s SqlProjectStore) RemoveAllMembersByProject(projectId string) StoreChannel {
	storeProject := make(StoreChannel)
	return storeProject
}

func (s SqlProjectStore) PermanentDeleteByProject(projectId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		if _, err := s.GetMaster().Exec("DELETE FROM Projects WHERE ProjectId = :ProjectId", map[string]interface{}{"ProjectId": projectId}); err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.PermanentDeleteByProject", "store.sql_project.permanent_delete_by_project.app_error", nil, "projectId="+projectId+", "+err.Error())
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

type projectWithMember struct {
	model.Project
	model.ProjectMember
}

func (s SqlProjectStore) GetProjects(teamId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		/*
		 *        var data []projectWithMember
		 *        _, err := s.GetReplica().Select(&data, "SELECT * FROM Projects, ProjectMembers WHERE Id = ProjectId AND UserId = :UserId AND DeleteAt = 0 AND (TeamId = :TeamId OR TeamId = '') ORDER BY DisplayName", map[string]interface{}{"TeamId": teamId, "UserId": userId})
		 *
		 *        if err != nil {
		 *            result.Err = model.NewLocAppError("SqlProjectStore.GetProjects", "store.sql_project.get_projects.get.app_error", nil, "teamId="+teamId+", userId="+userId+", err="+err.Error())
		 *        } else {
		 *            projects := &model.ProjectList{make([]*model.Project, len(data)), make(map[string]*model.ProjectMember)}
		 *            for i := range data {
		 *                v := data[i]
		 *                projects.Projects[i] = &v.Project
		 *                projects.Members[v.Project.Id] = &v.ProjectMember
		 *            }
		 *
		 *            if len(projects.Projects) == 0 {
		 *                result.Err = model.NewLocAppError("SqlProjectStore.GetProjects", "store.sql_project.get_projects.not_found.app_error", nil, "teamId="+teamId+", userId="+userId)
		 *            } else {
		 *                result.Data = projects
		 *            }
		 *        }
		 */

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) GetMoreProjects(teamId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		/*
		 *        var data []*model.Project
		 *        _, err := s.GetReplica().Select(&data,
		 *            `SELECT
		 *                *
		 *            FROM
		 *                Projects
		 *            WHERE
		 *                TeamId = :TeamId1
		 *                    AND Type IN ('O')
		 *                    AND DeleteAt = 0
		 *                    AND Id NOT IN (SELECT
		 *                        Projects.Id
		 *                    FROM
		 *                        Projects,
		 *                        ProjectMembers
		 *                    WHERE
		 *                        Id = ProjectId
		 *                            AND TeamId = :TeamId2
		 *                            AND UserId = :UserId
		 *                            AND DeleteAt = 0)
		 *            ORDER BY DisplayName`,
		 *            map[string]interface{}{"TeamId1": teamId, "TeamId2": teamId, "UserId": userId})
		 *
		 *        if err != nil {
		 *            result.Err = model.NewLocAppError("SqlProjectStore.GetMoreProjects", "store.sql_project.get_more_projects.get.app_error", nil, "teamId="+teamId+", userId="+userId+", err="+err.Error())
		 *        } else {
		 *            result.Data = &model.ProjectList{data, make(map[string]*model.ProjectMember)}
		 *        }
		 */

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

type projectIdWithCountAndUpdateAt struct {
	Id            string
	TotalMsgCount int64
	UpdateAt      int64
}

func (s SqlProjectStore) GetProjectCounts(teamId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		/*
		 *        var data []projectIdWithCountAndUpdateAt
		 *        _, err := s.GetReplica().Select(&data, "SELECT Id, TotalMsgCount, UpdateAt FROM Projects WHERE Id IN (SELECT ProjectId FROM ProjectMembers WHERE UserId = :UserId) AND (TeamId = :TeamId OR TeamId = '') AND DeleteAt = 0 ORDER BY DisplayName", map[string]interface{}{"TeamId": teamId, "UserId": userId})
		 *
		 *        if err != nil {
		 *            result.Err = model.NewLocAppError("SqlProjectStore.GetProjectCounts", "store.sql_project.get_project_counts.get.app_error", nil, "teamId="+teamId+", userId="+userId+", err="+err.Error())
		 *        } else {
		 *            counts := &model.ProjectCounts{Counts: make(map[string]int64), UpdateTimes: make(map[string]int64)}
		 *            for i := range data {
		 *                v := data[i]
		 *                counts.Counts[v.Id] = v.TotalMsgCount
		 *                counts.UpdateTimes[v.Id] = v.UpdateAt
		 *            }
		 *
		 *            result.Data = counts
		 *        }
		 */

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) GetByName(teamId string, name string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		//project := model.Project{}

		/*
		 *if err := s.GetReplica().SelectOne(&project, "SELECT * FROM Projects WHERE (TeamId = :TeamId OR TeamId = '') AND Name = :Name AND DeleteAt = 0", map[string]interface{}{"TeamId": teamId, "Name": name}); err != nil {
		 *    if err == sql.ErrNoRows {
		 *        result.Err = model.NewLocAppError("SqlProjectStore.GetByName", MISSING_project_ERROR, nil, "teamId="+teamId+", "+"name="+name+", "+err.Error())
		 *    } else {
		 *        result.Err = model.NewLocAppError("SqlProjectStore.GetByName", "store.sql_project.get_by_name.existing.app_error", nil, "teamId="+teamId+", "+"name="+name+", "+err.Error())
		 *    }
		 *} else {
		 *    result.Data = &project
		 *}
		 */

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) SaveMember(member *model.ProjectMember) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		var result StoreResult
		// Grab the project we are saving this member to
		if cr := <-s.GetFromMaster(member.ProjectId); cr.Err != nil {
			result.Err = cr.Err
		} else {
			project := cr.Data.(*model.Project)

			if transaction, err := s.GetMaster().Begin(); err != nil {
				result.Err = model.NewLocAppError("SqlProjectStore.SaveMember", "store.sql_project.save_member.open_transaction.app_error", nil, err.Error())
			} else {
				result = s.saveMemberT(transaction, member, project)
				if result.Err != nil {
					transaction.Rollback()
				} else {
					if err := transaction.Commit(); err != nil {
						result.Err = model.NewLocAppError("SqlProjectStore.SaveMember", "store.sql_project.save_member.commit_transaction.app_error", nil, err.Error())
					}
					// If sucessfull record members have changed in project
					if mu := <-s.extraUpdated(project); mu.Err != nil {
						result.Err = mu.Err
					}
				}
			}
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) saveMemberT(transaction *gorp.Transaction, member *model.ProjectMember, project *model.Project) StoreResult {
	result := StoreResult{}

	member.PreSave()
	if result.Err = member.IsValid(); result.Err != nil {
		return result
	}

	if err := transaction.Insert(member); err != nil {
		if IsUniqueConstraintError(err.Error(), []string{"ProjectId", "projectmembers_pkey"}) {
			result.Err = model.NewLocAppError("SqlProjectStore.SaveMember", "store.sql_project.save_member.exists.app_error", nil, "project_id="+member.ProjectId+", user_id="+member.UserId+", "+err.Error())
		} else {
			result.Err = model.NewLocAppError("SqlProjectStore.SaveMember", "store.sql_project.save_member.save.app_error", nil, "project_id="+member.ProjectId+", user_id="+member.UserId+", "+err.Error())
		}
	} else {
		result.Data = member
	}

	return result
}

func (s SqlProjectStore) UpdateMember(member *model.ProjectMember) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		member.PreUpdate()

		if result.Err = member.IsValid(); result.Err != nil {
			storeProject <- result
			close(storeProject)
			return
		}

		if _, err := s.GetMaster().Update(member); err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.UpdateMember", "store.sql_project.update_member.app_error", nil,
				"project_id="+member.ProjectId+", "+"user_id="+member.UserId+", "+err.Error())
		} else {
			result.Data = member
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) GetAllProjectListing() StoreChannel {
	storeProject := make(StoreChannel)
	return storeProject
}

func (s SqlProjectStore) GetMembers(projectId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		var members []model.ProjectMember
		_, err := s.GetReplica().Select(&members, "SELECT * FROM ProjectMembers WHERE ProjectId = :ProjectId", map[string]interface{}{"ProjectId": projectId})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.GetMembers", "store.sql_project.get_members.app_error", nil, "project_id="+projectId+err.Error())
		} else {
			result.Data = members
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) GetMember(projectId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		//var member model.ProjectMember

		/*
		 *if err := s.GetReplica().SelectOne(&member, "SELECT * FROM ProjectMembers WHERE ProjectId = :ProjectId AND UserId = :UserId", map[string]interface{}{"ProjectId": projectId, "UserId": userId}); err != nil {
		 *    if err == sql.ErrNoRows {
		 *        result.Err = model.NewLocAppError("SqlProjectStore.GetMember", MISSING_project_MEMBER_ERROR, nil, "project_id="+projectId+"user_id="+userId+","+err.Error())
		 *    } else {
		 *        result.Err = model.NewLocAppError("SqlProjectStore.GetMember", "store.sql_project.get_member.app_error", nil, "project_id="+projectId+"user_id="+userId+","+err.Error())
		 *    }
		 *} else {
		 *    result.Data = member
		 *}
		 */

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) GetMemberCount(projectId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		count, err := s.GetReplica().SelectInt(`
			SELECT
				count(*)
			FROM
				ProjectMembers,
				Users
			WHERE
				ProjectMembers.UserId = Users.Id
				AND ProjectMembers.ProjectId = :ProjectId
				AND Users.DeleteAt = 0`, map[string]interface{}{"ProjectId": projectId})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.GetMemberCount", "store.sql_project.get_member_count.app_error", nil, "project_id="+projectId+", "+err.Error())
		} else {
			result.Data = count
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) GetExtraMembers(projectId string, limit int) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		var members []model.ExtraMember
		var err error

		if limit != -1 {
			_, err = s.GetReplica().Select(&members, `
			SELECT
				Id,
				Nickname,
				Email,
				ProjectMembers.Roles,
				Username
			FROM
				ProjectMembers,
				Users
			WHERE
				ProjectMembers.UserId = Users.Id
				AND Users.DeleteAt = 0
				AND ProjectId = :ProjectId
			LIMIT :Limit`, map[string]interface{}{"ProjectId": projectId, "Limit": limit})
		} else {
			_, err = s.GetReplica().Select(&members, `
			SELECT
				Id,
				Nickname,
				Email,
				ProjectMembers.Roles,
				Username
			FROM
				ProjectMembers,
				Users
			WHERE
				ProjectMembers.UserId = Users.Id
				AND Users.DeleteAt = 0
				AND ProjectId = :ProjectId`, map[string]interface{}{"ProjectId": projectId})
		}

		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.GetExtraMembers", "store.sql_project.get_extra_members.app_error", nil, "project_id="+projectId+", "+err.Error())
		} else {
			for i := range members {
				members[i].Sanitize(utils.Cfg.GetSanitizeOptions())
			}
			result.Data = members
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) RemoveMember(projectId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		// Grab the project we are saving this member to
		if cr := <-s.Get(projectId); cr.Err != nil {
			result.Err = cr.Err
		} else {
			project := cr.Data.(*model.Project)

			_, err := s.GetMaster().Exec("DELETE FROM ProjectMembers WHERE ProjectId = :ProjectId AND UserId = :UserId", map[string]interface{}{"ProjectId": projectId, "UserId": userId})
			if err != nil {
				result.Err = model.NewLocAppError("SqlProjectStore.RemoveMember", "store.sql_project.remove_member.app_error", nil, "project_id="+projectId+", user_id="+userId+", "+err.Error())
			} else {
				// If sucessfull record members have changed in project
				if mu := <-s.extraUpdated(project); mu.Err != nil {
					result.Err = mu.Err
				}
			}
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) PermanentDeleteMembersByUser(userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		if _, err := s.GetMaster().Exec("DELETE FROM ProjectMembers WHERE UserId = :UserId", map[string]interface{}{"UserId": userId}); err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.RemoveMember", "store.sql_project.permanent_delete_members_by_user.app_error", nil, "user_id="+userId+", "+err.Error())
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) CheckPermissionsToNoTeam(projectId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		count, err := s.GetReplica().SelectInt(
			`SELECT
			    COUNT(0)
			FROM
			    Projects,
			    ProjectMembers
			WHERE
			    Projects.Id = ProjectMembers.ProjectId
			        AND Projects.DeleteAt = 0
			        AND ProjectMembers.ProjectId = :ProjectId
			        AND ProjectMembers.UserId = :UserId`,
			map[string]interface{}{"ProjectId": projectId, "UserId": userId})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.CheckPermissionsTo", "store.sql_project.check_permissions.app_error", nil, "project_id="+projectId+", user_id="+userId+", "+err.Error())
		} else {
			result.Data = count
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) CheckPermissionsTo(teamId string, projectId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		count, err := s.GetReplica().SelectInt(
			`SELECT
			    COUNT(0)
			FROM
			    Projects,
			    ProjectMembers
			WHERE
			    Projects.Id = ProjectMembers.ProjectId
			        AND (Projects.TeamId = :TeamId OR Projects.TeamId = '')
			        AND Projects.DeleteAt = 0
			        AND ProjectMembers.ProjectId = :ProjectId
			        AND ProjectMembers.UserId = :UserId`,
			map[string]interface{}{"TeamId": teamId, "ProjectId": projectId, "UserId": userId})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.CheckPermissionsTo", "store.sql_project.check_permissions.app_error", nil, "project_id="+projectId+", user_id="+userId+", "+err.Error())
		} else {
			result.Data = count
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) CheckPermissionsToByName(teamId string, projectName string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		projectId, err := s.GetReplica().SelectStr(
			`SELECT
			    Projects.Id
			FROM
			    Projects,
			    ProjectMembers
			WHERE
			    Projects.Id = ProjectMembers.ProjectId
			        AND (Projects.TeamId = :TeamId OR Projects.TeamId = '')
			        AND Projects.Name = :Name
			        AND Projects.DeleteAt = 0
			        AND ProjectMembers.UserId = :UserId`,
			map[string]interface{}{"TeamId": teamId, "Name": projectName, "UserId": userId})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.CheckPermissionsToByName", "store.sql_project.check_permissions_by_name.app_error", nil, "project_id="+projectName+", user_id="+userId+", "+err.Error())
		} else {
			result.Data = projectId
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) SetLastViewedAt(projectId string, userId string, newLastViewedAt int64) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		var query string

		if utils.Cfg.SqlSettings.DriverName == model.DATABASE_DRIVER_POSTGRES {
			query = `UPDATE
				ProjectMembers
			SET
			    MentionCount = 0,
			    MsgCount = Projects.TotalMsgCount - (SELECT COUNT(*)
			    					 FROM Posts
			    					 WHERE ProjectId = :ProjectId
			    					 AND CreateAt > :NewLastViewedAt),
			    LastViewedAt = :NewLastViewedAt
			FROM
				Projects
			WHERE
			    Projects.Id = ProjectMembers.ProjectId
			        AND UserId = :UserId
			        AND ProjectId = :ProjectId`
		} else if utils.Cfg.SqlSettings.DriverName == model.DATABASE_DRIVER_MYSQL {
			query = `UPDATE
				ProjectMembers, Projects
			SET
			    ProjectMembers.MentionCount = 0,
			    ProjectMembers.MsgCount = Projects.TotalMsgCount - (SELECT COUNT(*)
										FROM Posts
										WHERE ProjectId = :ProjectId
										AND CreateAt > :NewLastViewedAt),
			    ProjectMembers.LastViewedAt = :NewLastViewedAt
			WHERE
			    Projects.Id = ProjectMembers.ProjectId
			        AND UserId = :UserId
			        AND ProjectId = :ProjectId`
		}

		_, err := s.GetMaster().Exec(query, map[string]interface{}{"ProjectId": projectId, "UserId": userId, "NewLastViewedAt": newLastViewedAt})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.SetLastViewedAt", "store.sql_project.set_last_viewed_at.app_error", nil, "project_id="+projectId+", user_id="+userId+", "+err.Error())
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) UpdateLastViewedAt(projectId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		var query string

		if utils.Cfg.SqlSettings.DriverName == model.DATABASE_DRIVER_POSTGRES {
			query = `UPDATE
				ProjectMembers
			SET
			    MentionCount = 0,
			    MsgCount = Projects.TotalMsgCount,
			    LastViewedAt = Projects.LastPostAt,
			    LastUpdateAt = Projects.LastPostAt
			FROM
				Projects
			WHERE
			    Projects.Id = ProjectMembers.ProjectId
			        AND UserId = :UserId
			        AND ProjectId = :ProjectId`
		} else if utils.Cfg.SqlSettings.DriverName == model.DATABASE_DRIVER_MYSQL {
			query = `UPDATE
				ProjectMembers, Projects
			SET
			    ProjectMembers.MentionCount = 0,
			    ProjectMembers.MsgCount = Projects.TotalMsgCount,
			    ProjectMembers.LastViewedAt = Projects.LastPostAt,
			    ProjectMembers.LastUpdateAt = Projects.LastPostAt
			WHERE
			    Projects.Id = ProjectMembers.ProjectId
			        AND UserId = :UserId
			        AND ProjectId = :ProjectId`
		}

		_, err := s.GetMaster().Exec(query, map[string]interface{}{"ProjectId": projectId, "UserId": userId})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.UpdateLastViewedAt", "store.sql_project.update_last_viewed_at.app_error", nil, "project_id="+projectId+", user_id="+userId+", "+err.Error())
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) IncrementMentionCount(projectId string, userId string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		_, err := s.GetMaster().Exec(
			`UPDATE
				ProjectMembers
			SET
				MentionCount = MentionCount + 1
			WHERE
				UserId = :UserId
					AND ProjectId = :ProjectId`,
			map[string]interface{}{"ProjectId": projectId, "UserId": userId})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.IncrementMentionCount", "store.sql_project.increment_mention_count.app_error", nil, "project_id="+projectId+", user_id="+userId+", "+err.Error())
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) GetAll() StoreChannel {
	storeProject := make(StoreChannel)

	//go func() {
	//result := StoreResult{}

	//var data []*model.Project
	//_, err := s.GetReplica().Select(&data, "SELECT * FROM Projects WHERE TeamId = :TeamId AND Type != 'D' ORDER BY Name", map[string]interface{}{"TeamId": teamId})

	//if err != nil {
	//result.Err = model.NewLocAppError("SqlProjectStore.GetAll", "store.sql_project.get_all.app_error", nil, "teamId="+teamId+", err="+err.Error())
	//} else {
	//result.Data = data
	//}

	//storeProject <- result
	//close(storeProject)
	//}()

	return storeProject
}

func (s SqlProjectStore) GetByInviteId(inviteId string) StoreChannel {
	storeProject := make(StoreChannel)
	return storeProject
}

func (s SqlProjectStore) AnalyticsTypeCount(teamId string, projectType string) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		query := "SELECT COUNT(Id) AS Value FROM Projects WHERE Type = :ProjectType"

		if len(teamId) > 0 {
			query += " AND TeamId = :TeamId"
		}

		v, err := s.GetReplica().SelectInt(query, map[string]interface{}{"TeamId": teamId, "ProjectType": projectType})
		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.AnalyticsTypeCount", "store.sql_project.analytics_type_count.app_error", nil, err.Error())
		} else {
			result.Data = v
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}

func (s SqlProjectStore) ExtraUpdateByUser(userId string, time int64) StoreChannel {
	storeProject := make(StoreChannel)

	go func() {
		result := StoreResult{}

		_, err := s.GetMaster().Exec(
			`UPDATE Projects SET ExtraUpdateAt = :Time
			WHERE Id IN (SELECT ProjectId FROM ProjectMembers WHERE UserId = :UserId);`,
			map[string]interface{}{"UserId": userId, "Time": time})

		if err != nil {
			result.Err = model.NewLocAppError("SqlProjectStore.extraUpdated", "store.sql_project.extra_updated.app_error", nil, "user_id="+userId+", "+err.Error())
		}

		storeProject <- result
		close(storeProject)
	}()

	return storeProject
}
