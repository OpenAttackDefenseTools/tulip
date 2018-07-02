// @flow

import React, { Component } from "react";

//ui
import { withStyles } from "@material-ui/core/styles";
import InputLabel from "@material-ui/core/InputLabel";
import MenuItem from "@material-ui/core/MenuItem";
import FormControl from "@material-ui/core/FormControl";
import Select from "@material-ui/core/Select";

import { fetchServices } from "../data/fetcher";

const styles = theme => ({
    root: {
        display: "flex",
        flexWrap: "wrap"
    },
    formControl: {
        margin: 0,
        padding: 2,
        minWidth: 120
    },
    selectEmpty: {
        marginTop: theme.spacing.unit * 2
    }
});
//{ip: "127.0.0.1", port: 80, name: "first service"}
type Service_type={
    ip: string,
    port: number,
    name: string
};
type props_types = {
    onServicesFetched: (Array<Service_type>)=>void,
    classes:*
};
type state_types={
    services : Array<Service_type>,
    target_name: string
};
export class ServiceSelector extends Component<props_types,state_types> {
    constructor(props:props_types) {
        super(props);
        this.state = {
            services: [],
            target_name: "All"
        };
    }
    componentDidMount() {
        fetchServices(services => {
            console.log("ok, ho i servizi: ");
            console.log(services);
            this.setState({ services: services });
            this.props.onServicesFetched(services)
        });
    }
    render() {
        const { classes } = this.props;
        console.log("nome che dovriei mettere: " + this.state.target_name);
        return (
            <form className={classes.root} autoComplete="off">
                <FormControl className={classes.formControl}>
                    <InputLabel htmlFor="target_name">Service</InputLabel>
                    <Select
                        value={this.state.target_name}
                        onChange={this.handleChange}
                        inputProps={{
                            name: "target_name",
                            id: "target_name-simple"
                        }}
                    >
                        <MenuItem value={-1}>
                            <em>All</em>
                        </MenuItem>

                        {this.state.services.map((item, inx) => (
                            <MenuItem key={item.name} value={inx}>
                                <em>{item.name}</em>
                            </MenuItem>
                        ))}
                    </Select>
                </FormControl>
            </form>
        );
    }
    handleChange = event => {
        this.setState({ [event.target.name]: event.target.value });

        var inx = event.target.value;
        if (inx === -1) this.props.onTargetChanged(null);
        else {
            var target_service = this.state.services[inx];
            console.log("target service:")
            console.log(target_service );
            this.props.onTargetChanged(target_service.ip,target_service.port);
        }
    };
}
export default withStyles(styles)(ServiceSelector);
