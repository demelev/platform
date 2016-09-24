// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import AppDispatcher from '../dispatcher/app_dispatcher.jsx';
import EventEmitter from 'events';
import ChannelStore from 'stores/channel_store.jsx';
import Client from 'client/web_client.jsx';
import * as AsyncClient from 'utils/async_client.jsx';

var Utils;
import Constants from 'utils/constants.jsx';
const ActionTypes = Constants.ActionTypes;

//const NotificationPrefs = Constants.NotificationPrefs;

const CHANGE_EVENT = 'change';
const LEAVE_EVENT = 'leave';
const MORE_CHANGE_EVENT = 'change';
const EXTRA_INFO_EVENT = 'extra_info';
const LAST_VIEVED_EVENT = 'last_viewed';

class ProjectStoreClass extends EventEmitter {
    constructor(props) {
        super(props);

        this.setMaxListeners(15);

        this.emitChange = this.emitChange.bind(this);
        this.addChangeListener = this.addChangeListener.bind(this);
        this.removeChangeListener = this.removeChangeListener.bind(this);
        this.emitMoreChange = this.emitMoreChange.bind(this);
        this.addMoreChangeListener = this.addMoreChangeListener.bind(this);
        this.removeMoreChangeListener = this.removeMoreChangeListener.bind(this);
        this.emitExtraInfoChange = this.emitExtraInfoChange.bind(this);
        this.addExtraInfoChangeListener = this.addExtraInfoChangeListener.bind(this);
        this.removeExtraInfoChangeListener = this.removeExtraInfoChangeListener.bind(this);
        this.emitLeave = this.emitLeave.bind(this);
        this.addLeaveListener = this.addLeaveListener.bind(this);
        this.removeLeaveListener = this.removeLeaveListener.bind(this);
        this.emitLastViewed = this.emitLastViewed.bind(this);
        this.addLastViewedListener = this.addLastViewedListener.bind(this);
        this.removeLastViewedListener = this.removeLastViewedListener.bind(this);
        this.findFirstBy = this.findFirstBy.bind(this);
        this.get = this.get.bind(this);
        this.getMember = this.getMember.bind(this);
        this.getByName = this.getByName.bind(this);
        this.getByDisplayName = this.getByDisplayName.bind(this);
        this.setPostMode = this.setPostMode.bind(this);
        this.getPostMode = this.getPostMode.bind(this);
        this.setUnreadCount = this.setUnreadCount.bind(this);
        this.setUnreadCounts = this.setUnreadCounts.bind(this);
        this.getUnreadCount = this.getUnreadCount.bind(this);
        this.getUnreadCounts = this.getUnreadCounts.bind(this);

        this.currentId = null;
        this.postMode = this.POST_MODE_PROJECT;
        this.projects = [];
        this.projectMembers = {};
        this.projectChannels = {};
        this.moreProjects = {};
        this.moreProjects.loading = true;
        this.extraInfos = {};
        this.unreadCounts = {};
    }

    get POST_MODE_PROJECT() {
        return 1;
    }

    get POST_MODE_FOCUS() {
        return 2;
    }

    emitChange() {
        this.emit(CHANGE_EVENT);
    }

    addChangeListener(callback) {
        this.on(CHANGE_EVENT, callback);
    }

    removeChangeListener(callback) {
        this.removeListener(CHANGE_EVENT, callback);
    }

    emitMoreChange() {
        this.emit(MORE_CHANGE_EVENT);
    }

    addMoreChangeListener(callback) {
        this.on(MORE_CHANGE_EVENT, callback);
    }

    removeMoreChangeListener(callback) {
        this.removeListener(MORE_CHANGE_EVENT, callback);
    }

    emitExtraInfoChange() {
        this.emit(EXTRA_INFO_EVENT);
    }

    addExtraInfoChangeListener(callback) {
        this.on(EXTRA_INFO_EVENT, callback);
    }

    removeExtraInfoChangeListener(callback) {
        this.removeListener(EXTRA_INFO_EVENT, callback);
    }
    emitLeave(id) {
        this.emit(LEAVE_EVENT, id);
    }

    addLeaveListener(callback) {
        this.on(LEAVE_EVENT, callback);
    }

    removeLeaveListener(callback) {
        this.removeListener(LEAVE_EVENT, callback);
    }

    emitLastViewed(lastViewed, ownNewMessage) {
        this.emit(LAST_VIEVED_EVENT, lastViewed, ownNewMessage);
    }

    addLastViewedListener(callback) {
        this.on(LAST_VIEVED_EVENT, callback);
    }

    removeLastViewedListener(callback) {
        this.removeListener(LAST_VIEVED_EVENT, callback);
    }

    findFirstBy(field, value) {
        return this.doFindFirst(field, value, this.getProjects());
    }

    findFirstMoreBy(field, value) {
        return this.doFindFirst(field, value, this.getMoreProjects());
    }

    doFindFirst(field, value, projects) {
        for (var i = 0; i < projects.length; i++) {
            if (projects[i][field] === value) {
                return projects[i];
            }
        }

        return null;
    }

    get(id) {
        return this.findFirstBy('id', id);
    }

    getMember(id) {
        return this.getAllMembers()[id];
    }

    getByName(name) {
        return this.findFirstBy('name', name);
    }

    getByDisplayName(displayName) {
        return this.findFirstBy('display_name', displayName);
    }

    getMoreByName(name) {
        return this.findFirstMoreBy('name', name);
    }

    getAll() {
        return this.getProjects();
    }

    getAllMembers() {
        return this.getProjectMembers();
    }

    getMoreAll() {
        return this.getMoreProjects();
    }

    setCurrentId(id) {
        this.currentId = id;
    }

    resetCounts(id) {
        const cm = this.projectMembers;
        for (var cmid in cm) {
            if (cm[cmid].project_id === id) {
                var c = this.get(id);
                if (c) {
                    cm[cmid].msg_count = this.get(id).total_msg_count;
                    cm[cmid].mention_count = 0;
                    this.setUnreadCount(id);
                }
                break;
            }
        }
    }

    getCurrentId() {
        return this.currentId;
    }

    getCurrent() {
        var currentId = this.getCurrentId();

        if (currentId) {
            return this.get(currentId);
        }

        return null;
    }

    getCurrentMember() {
        var currentId = this.getCurrentId();

        if (currentId) {
            return this.getAllMembers()[currentId];
        }

        return null;
    }

    setProjectMember(member) {
        var members = this.getProjectMembers();
        members[member.project_id] = member;
        this.storeProjectMembers(members);
        this.emitChange();
    }

    getCurrentExtraInfo() {
        return this.getExtraInfo(this.getCurrentId());
    }

    getExtraInfo(projectId) {
        var extra = null;

        if (projectId) {
            extra = this.getExtraInfos()[projectId];
        }

        if (extra) {
            // create a defensive copy
            extra = JSON.parse(JSON.stringify(extra));
        } else {
            extra = {members: []};
        }

        return extra;
    }

    pStoreProject(project) {
        var projects = this.getProjects();
        var found;

        for (var i = 0; i < projects.length; i++) {
            if (projects[i].id === project.id) {
                projects[i] = project;
                found = true;
                break;
            }
        }

        if (!found) {
            projects.push(project);
        }

        if (!Utils) {
            Utils = require('utils/utils.jsx'); //eslint-disable-line global-require
        }

        projects.sort(Utils.sortByDisplayName);
        this.storeProjects(projects);
    }

    storeProjects(projects) {
        this.projects = projects;
    }

    getProjects() {
        return this.projects;
    }

    pStoreProjectMember(projectMember) {
        var members = this.getProjectMembers();
        members[projectMember.project_id] = projectMember;
        this.storeProjectMembers(members);
    }

    storeProjectMembers(projectMembers) {
        this.projectMembers = projectMembers;
    }

    storeProjectChannels(projectChannels) {
        this.projectChannels = projectChannels;
        Client.logClientError("ProjectChannels store : " + JSON.stringify(projectChannels));
    }

    getProjectMembers() {
        return this.projectMembers;
    }

    getProjectsChannels() {
        return this.projectChannels;
    }

    getProjectChannels(id) {
        return this.projectChannels[id];
    }

    getCurrentChannels() {
        var id = this.getCurrentId();
        return this.getProjectChannels(id);
    }

    storeMoreProjects(projects) {
        this.moreProjects = projects;
    }

    getMoreProjects() {
        return this.moreProjects;
    }

    storeExtraInfos(extraInfos) {
        this.extraInfos = extraInfos;
    }

    getExtraInfos() {
        return this.extraInfos;
    }

    isDefault(project) {
        return project.name === Constants.DEFAULT_PROJECT;
    }

    setPostMode(mode) {
        this.postMode = mode;
    }

    getPostMode() {
        return this.postMode;
    }

    setUnreadCount(id) {
        //const ch = this.get(id);
        //const chMember = this.getMember(id);

        //let chMentionCount = chMember.mention_count;
        //let chUnreadCount = ch.total_msg_count - chMember.msg_count - chMentionCount;

        //if (ch.type === 'D') {
            //chMentionCount = chUnreadCount;
            //chUnreadCount = 0;
        //} else if (chMember.notify_props && chMember.notify_props.mark_unread === NotificationPrefs.MENTION) {
            //chUnreadCount = 0;
        //}

        //this.unreadCounts[id] = {msgs: chUnreadCount, mentions: chMentionCount};
    }

    setUnreadCounts() {
        const projects = this.getAll();
        projects.forEach((ch) => {
            this.setUnreadCount(ch.id);
        });
    }

    getUnreadCount(id) {
        return this.unreadCounts[id] || {msgs: 0, mentions: 0};
    }

    getUnreadCounts() {
        return this.unreadCounts;
    }

    leaveProject(id) {
        Reflect.deleteProperty(this.projectMembers, id);
        const element = this.projects.indexOf(id);
        if (element > -1) {
            this.projects.splice(element, 1);
        }
    }
}

var ProjectStore = new ProjectStoreClass();

ProjectStore.dispatchToken = AppDispatcher.register((payload) => {
    var action = payload.action;
    var currentId;

    switch (action.type) {
    case ActionTypes.CLICK_PROJECT:
        ProjectStore.setCurrentId(action.id);
        var channelsId = ProjectStore.getCurrentChannels();

        if (channelsId == null) {
            Client.logClientError('ProjectStore: CLICK_PROJECT: There is no channelsId');
        } else {
            ChannelStore.setCurrentId(channelsId[0]);
            AsyncClient.getChannelExtraInfo(channelsId[0]);
            ChannelStore.emitChange();
        }

        ProjectStore.emitChange();

        break;

    case ActionTypes.RECEIVED_PROJECTS:
        ProjectStore.storeProjects(action.projects);
        ProjectStore.storeProjectMembers(action.members);
        currentId = ProjectStore.getCurrentId();

        //if (currentId && window.isActive) {
            //ProjectStore.resetCounts(currentId);
        //}
        //ProjectStore.setUnreadCounts();
        ProjectStore.emitChange();
        break;

    case ActionTypes.RECEIVED_PROJECTS_CHANNELS:
        Client.logClientError('>>> Channels id is : ' + JSON.stringify(action));
        ProjectStore.storeProjectChannels(action.channels_id);
        ProjectStore.emitChange();
        break;

    case ActionTypes.RECEIVED_PROJECT:
        ProjectStore.pStoreProject(action.project);
        if (action.member) {
            ProjectStore.pStoreProjectMember(action.member);
        }
        currentId = ProjectStore.getCurrentId();
        if (currentId && window.isActive) {
            ProjectStore.resetCounts(currentId);
        }
        ProjectStore.setUnreadCount(action.project.id);
        ProjectStore.emitChange();
        break;

    case ActionTypes.RECEIVED_MORE_PROJECTS:
        ProjectStore.storeMoreProjects(action.projects);
        ProjectStore.emitMoreChange();
        break;

    case ActionTypes.RECEIVED_PROJECT_EXTRA_INFO:
        var extraInfos = ProjectStore.getExtraInfos();
        extraInfos[action.extra_info.id] = action.extra_info;
        ProjectStore.storeExtraInfos(extraInfos);
        ProjectStore.emitExtraInfoChange();
        break;

    case ActionTypes.LEAVE_PROJECT:
        ProjectStore.leaveProject(action.id);
        ProjectStore.emitLeave(action.id);
        break;

    default:
        break;
    }
});

export default ProjectStore;
