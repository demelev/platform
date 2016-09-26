// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import $ from 'jquery';
import React from 'react';

//import ProjectHeader from 'components/project_header.jsx';
import ChannelHeader from 'components/channel_header.jsx';
import FileUploadOverlay from 'components/file_upload_overlay.jsx';
import CreatePost from 'components/create_post.jsx';
import PostViewCache from 'components/post_view/post_view_cache.jsx';
import Client from 'client/web_client.jsx';

import ChannelStore from 'stores/channel_store.jsx';
import ProjectStore from 'stores/project_store.jsx';

import * as Utils from 'utils/utils.jsx';

export default class ProjectView extends React.Component {
    constructor(props) {
        super(props);

        this.getStateFromStores = this.getStateFromStores.bind(this);
        this.isStateValid = this.isStateValid.bind(this);
        this.updateState = this.updateState.bind(this);

        this.state = this.getStateFromStores(props);
    }

    getStateFromStores(props) {
        const project = ProjectStore.getByName(props.params.project);
        const channels = ProjectStore.getProjectChannels(project.id);
        var channel = null;

        if (channels)
            channel = ChannelStore.get(channels[0]); //TODO: implement it as it should be

        const channelId = channel ? channel.id : '';
        return {
            channelId
        };
    }
    isStateValid() {
        return this.state.channelId !== '';
    }
    updateState() {
        this.setState(this.getStateFromStores(this.props));
    }
    componentDidMount() {
        ProjectStore.addChangeListener(this.updateState);

        $('body').addClass('app__body');
    }
    componentWillUnmount() {
        ProjectStore.removeChangeListener(this.updateState);

        $('body').removeClass('app__body');
    }
    componentWillReceiveProps(nextProps) {
        this.setState(this.getStateFromStores(nextProps));
    }
    shouldComponentUpdate(nextProps, nextState) {
        //Client.logClientError('DEBUG: params compare ' + JSON.stringify(nextProps.params) + ' and ' + JSON.stringify(this.props.params));
        if (!Utils.areObjectsEqual(nextProps.params, this.props.params)) {
            return true;
        }

        //Client.logClientError('DEBUG: channel id compare: ' + nextState.channelId + ' and ' + this.state.channelId);
        if (nextState.channelId !== this.state.channelId) {
            return true;
        }

        return false;
    }
    render() {
        return (
            <div
                id='app-content'
                className='app__content'
            >
                <FileUploadOverlay overlayType='center'/>
                <ChannelHeader
                    channelId={this.state.channelId}
                />
                <PostViewCache/>
                <div
                    className='post-create__container'
                    id='post-create'
                >
                    <CreatePost/>
                </div>
            </div>
        );
    }
}
ProjectView.defaultProps = {
};

ProjectView.propTypes = {
    params: React.PropTypes.object.isRequired
};
