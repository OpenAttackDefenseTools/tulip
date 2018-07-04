/* 
 * This file is part of Flower.
 * 
 * Copyright ©2018 Nicolò Mazzucato
 * Copyright ©2018 Antonio Groza
 * Copyright ©2018 Brunello Simone
 * Copyright ©2018 Alessio Marotta
 * DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS FILE HEADER.
 * 
 * Flower is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 * 
 * Flower is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * 
 * You should have received a copy of the GNU General Public License
 * along with Flower.  If not, see <https://www.gnu.org/licenses/>.
 */

// @flow

import React, { Component } from "react";

//ui
import { withStyles } from "@material-ui/core/styles";
import ListItem from "@material-ui/core/ListItem";
import Divider from "@material-ui/core/Divider";
import ListItemIcon from "@material-ui/core/ListItemIcon";
import Grid from "@material-ui/core/Grid";
import Checkbox from "@material-ui/core/Checkbox";
import Favorite from "@material-ui/icons/Favorite";
import FavoriteBorder from "@material-ui/icons/FavoriteBorder";

import moment from "moment";

import { setStarred } from "../data/fetcher.js";
const styles = theme => ({
    root: {
        maxWidth: 360,
        paddingTop: 0
    }
});
type FlowItem_type = {
    _id: *,
    inx: number,
    time: number,
    duration: number,
    src_ip: string,
    src_port: number,
    dst_ip: string,
    dst_port: number,
    contains_flag: boolean,
    starred: boolean,
    flow: Array<*>
};
type props_types = {
    item: FlowItem_type,
    selected: boolean,
    hideFavourite: boolean,
    large: boolean,
    serviceName: string,
    classes: *, // material ui things
    onClick: FlowItem_type => void,
    onStar: (boolean) => void
};
type state_types = {
    starred: boolean
};
export class FlowItem extends Component<props_types, state_types> {
    constructor(props: props_types) {
        super(props);
        this.state = { starred: props.item.starred || false };
    }
    render() {
        const { item } = this.props;

        const item_color = this.getItemColor(item);
        return (
            <div style={{ fontSize: "15px" }}>
                <ListItem
                    style={{
                        backgroundColor: item_color,
                        display: "table",
                        width: "100%"
                    }}
                    button
                    onClick={() =>
                        this.props.onClick && this.props.onClick(item)
                    }
                >
                    {this.renderItem(item)}
                </ListItem>
                <Divider />
            </div>
        );
    }

    renderItem(item: FlowItem_type) {
        const checked =
            item.starred == null ? false : item.starred ? true : false;

        return (
            <Grid container alignItems="center" direction="row">
                {this.props.hideFavourite || (
                    <ListItemIcon>
                        <Checkbox
                            icon={<FavoriteBorder />}
                            checkedIcon={<Favorite />}
                            checked={checked}
                            onClick={e => {
                                setStarred(item._id["$oid"], !item.starred);
                                this.props.onStar &&
                                    this.props.onStar(!item.starred);
                                e.stopPropagation();
                            }}
                        />
                    </ListItemIcon>
                )}
                <div>
                    <b>{this.props.serviceName || item.dst_port}</b>
                    {this.props.large && this.getIpSourceDestInfo()}
                    {this.getTimeInfo()}
                </div>
            </Grid>
        );
    }

    getItemColor(item: FlowItem_type) {
        const isSelected = this.props.selected || false;
        if (isSelected) return "#03a9f4";
        if (item.contains_flag) return "#FF6666";
        return "#F5F5F5";
    }
    getIpSourceDestInfo() {
        const item = this.props.item;
        return (
            <div>
                {item.src_ip + ":"}
                <b>{item.src_port}</b> {"⇨ " + item.dst_ip + ":"}
                <font color="red">
                    <b>{item.dst_port}</b>
                </font>
            </div>
        );
    }
    getTimeInfo() {
        const item = this.props.item;
        return (
            <div>
                {moment(item.time).format("HH:mm:ss,")}
                <b>{moment(item.time).format("SSS")}</b>
                {"ms "}
                {this.props.large && " duration: " + item.duration + "ms"}
            </div>
        );
    }
}
export default withStyles(styles)(FlowItem);
