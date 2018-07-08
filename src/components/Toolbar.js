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
import Toolbar from "@material-ui/core/Toolbar";
import SearchBar from "material-ui-search-bar";
import Typography from "@material-ui/core/Typography";
import CircularProgress from "@material-ui/core/CircularProgress";
import Checkbox from "@material-ui/core/Checkbox";
import FormControlLabel from "@material-ui/core/FormControlLabel";
import TextField from "@material-ui/core/TextField";

import { withStyles } from "@material-ui/core/styles";
import IconButton from "@material-ui/core/IconButton";

import ServiceSelector from "./ServiceSelector";
import Service_type from "./ServiceSelector";
const styles = theme => ({
   appBar: {
      zIndex: theme.zIndex.drawer + 1,
      position: "fixed"
   },
   toolbar: theme.mixins.toolbar,
   progress: {
      margin: theme.spacing.unit
   },
   textField: {
      marginLeft: theme.spacing.unit,
      marginRight: theme.spacing.unit,
      width: 100,
      borderRadius: 2,
      padding: 3,
      backgroundColor: "#FFFFFF"
   },
   serviceSelector: {
      marginLeft: theme.spacing.unit,
      marginRight: theme.spacing.unit,
      borderRadius: 2,
      backgroundColor: "#FFFFFF"
   }
});
type props_types = {
   classes: *,
   onTargetChanged: (*) => void,
   hexdump: boolean,
   requestInProgress: boolean,
   onRequestSearch: string => void,
   toggleHexdump: boolean => void,
   onTimeSet: (number, number) => void,
   onServicesFetched: (Array<Service_type>) => void
};
type state_types = {
   from_time: number,
   to_time: number
};
export class MyToolbar extends Component<props_types, state_types> {
   constructor(props: props_types) {
      super(props);
      this.state = {
         from_time: 0,
         to_time: Number.MAX_SAFE_INTEGER
      };
   }
   render() {
      const { classes } = this.props;

      return (
         <Toolbar>
            <IconButton color="inherit" aria-label="Menu" />

            {/*// eslint-disable-next-line*/}
            <Typography variant="title" color="inherit">
               Flower
            </Typography>
            <span role="img" aria-label="flower" style={{margin:10, fontSize:30}}>🌸</span>
            <SearchBar
               onRequestSearch={this.props.onRequestSearch}
               style={{
                  margin: 10,
                  padding: 3,
                  marginLeft: 100,
                  maxWidth: 400
               }}
            />
            <TextField
               id="from_time"
               label="From"
               type="time"
               defaultValue="00:00"
               className={classes.textField}
               InputLabelProps={{
                  shrink: true
               }}
               inputProps={{
                  step: 300 // 5 min
               }}
               onChange={item => {
                  var time = this.getTimeFromString(item.target.value);
                  console.log(item.target.value);
                  this.setState({ from_time: time });
                  this.props.onTimeSet(time, this.state.to_time);
               }}
            />
            <TextField
               id="to_time"
               label="To"
               type="time"
               defaultValue={this.getActualTimeString()}
               className={classes.textField}
               InputLabelProps={{
                  shrink: true
               }}
               inputProps={{
                  step: 300 // 5 min
               }}
               onChange={item => {
                  var time = this.getTimeFromString(item.target.value);
                  if (item === 0) item = Number.MAX_SAFE_INTEGER;
                  console.log(item.target.value);
                  this.setState({ to_time: time });
                  this.props.onTimeSet(this.state.from_time, time);
               }}
            />
            <div className={classes.serviceSelector}>
               <ServiceSelector
                  onTargetChanged={this.props.onTargetChanged}
                  onServicesFetched={this.props.onServicesFetched}
                  style={{ margin: 0, padding: 0 }}
               />
            </div>

            <FormControlLabel
               control={
                  <Checkbox
                     checked={this.props.hexdump}
                     onChange={this.props.toggleHexdump}
                     value="Hexdump"
                  />
               }
               className={classes.checkbox}
               label="Hexdump"
               style={{ margin: 5 }}
            />

            {this.props.requestInProgress ? (
               <CircularProgress
                  className={classes.progress}
                  color="secondary"
               />
            ) : null}
         </Toolbar>
      );
   }

   getActualTimeString() {
      var d = new Date();
      return d.getHours() + ":" + d.getMinutes();
   }
   getTimeFromString(time_str: string) {
      if (time_str.length === 0) return 0;
      var h = parseInt(time_str.split(":")[0],10);
      var min = parseInt(time_str.split(":")[1],10);
      var d = new Date();
      d.setUTCHours(h - 2); //fast time-zone fix TODO FIX THIS
      d.setUTCMinutes(min);
      return d.getTime();
   }
}
export default withStyles(styles)(MyToolbar);
