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

import { withStyles } from "@material-ui/core/styles";
import { List, AutoSizer } from "react-virtualized";

import FlowItem from "./FlowItem";
import FlowItem_type from "./FlowItem";
import { fetchFlows } from "../data/fetcher";
//import VirtualList from "react-tiny-virtual-list";

import _ from "lodash/core";

const styles = theme => ({
    root: {
        overflow: "auto",
        maxWidth: 360,
        backgroundColor: theme.palette.background.paper
    }
});

type state_types = {
    data: Array<FlowItem_type>,
    filters_applied: *,
    selected_inx: number
};
type props_types = {
    largeItems: boolean,
    filters: *,
    onFlowsLoaded: () => void,
    onFlowSelected: FlowItem_type => void,
    list_id: string,
    classes: *, //material ui things
    services: Array<*>
};
export class FlowList extends Component<props_types, state_types> {
    constructor(props: props_types) {
        super(props);
        this.state = { data: [], filters_applied: {}, selected_inx: 0 };
    }
    componentDidMount() {
        this.loadFlows();
    }
    componentDidUpdate() {
        this.loadFlows();
    }

    loadFlows() {
        var filters = this.props.filters;
        //teniamo solo le chiavi che ci interessano
        //todo non usare testo hardcoded
        var prop = [
            "text_filter",
            "dst_ip",
            "dst_port",
            "from_time",
            "to_time",
            "starred"
        ];
        for (var k in filters) {
            if (prop.indexOf(k) < 0) {
                delete filters[k];
            }
        }

        if (_.isEqual(filters, this.state.filters_applied)) return;
        this.setState({
            filters_applied: filters
        });
        fetchFlows(filters, flows => {
            console.log("ok, ho i flows!: ");
            console.log(flows);
            this.setState({
                data: flows,
                filters_applied: filters
            });
            this.props.onFlowsLoaded();
        });
    }

    renderRow = ({ index, key, style }) => {
        let item = this.state.data[index];
        return (
            <div key={key} style={style} className="row">
                <FlowItem
                    selected={index === this.state.selected_inx}
                    item={item}
                    serviceName={this.getServiceName(item)}
                    key={item._id["$oid"] + this.props.list_id}
                    onClick={flow => {
                        this.setState({ selected_inx: index });
                        this.props.onFlowSelected(flow);
                    }}
                    onStar={star => {
                        console.log("Cambio stato stella!");
                        let data = [...this.state.data];
                        data[index].starred = star; //new value
                        this.setState({ data });
                    }}
                    large={this.props.largeItems}
                />
            </div>
        );
    };

    handleKeyPress = (e: KeyboardEvent<HTMLDivElement>) => {
        console.log("premuto:" + e.key);
        if (e.key === "k") {
            console.log("up key detected.");
            if (this.state.selected_inx === 0) return;
            this.setState(prevState => ({
                selected_inx: prevState.selected_inx - 1
            }));
            this.props.onFlowSelected(
                this.state.data[this.state.selected_inx - 1]
            );
            this.list.scrollToRow(this.state.selected_inx);
        } else if (e.key === "j") {
            console.log("down key detected.");
            if (this.state.selected_inx === this.state.data.length - 1) return;
            this.setState(prevState => ({
                selected_inx: prevState.selected_inx + 1
            }));
            this.props.onFlowSelected(
                this.state.data[this.state.selected_inx + 1]
            );
            this.list.scrollToRow(this.state.selected_inx + 2);
        }
    };

    render() {

        var data = this.state.data;
        console.log("richiamato render flow list");

        return (
            <AutoSizer data={this.state.data}>
                {({ height, width }) => (
                    <div onKeyPress={this.handleKeyPress}>
                        <List
                            data={this.state.data}
                            index={this.state.selected_inx}
                            width={width}
                            height={height}
                            rowHeight={80}
                            ref={(list: *) => {
                                this.list = list;
                            }}
                            rowRenderer={this.renderRow}
                            rowCount={data.length}
                        />
                    </div>
                )}
            </AutoSizer>
        );
    }

    getServiceName(item: FlowItem_type) {
        var port = item.dst_port;
        for (var service of this.props.services) {
            if (service.port === port) return service.name;
        }
        return "unknown";
    }
}

export default withStyles(styles)(FlowList);
