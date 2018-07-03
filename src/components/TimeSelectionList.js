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
import List from "@material-ui/core/List";
import ListItem from "@material-ui/core/ListItem";
import ListItemIcon from "@material-ui/core/ListItemIcon";
import ListItemText from "@material-ui/core/ListItemText";
import Divider from "@material-ui/core/Divider";

//utility
import { fetchFiles } from "../data/fetcher";
import TimeRangePicker from "react-time-range-picker";
import moment from "moment";

const styles = theme => ({
   root: {
      width: "100%",
      backgroundColor: theme.palette.background.paper
   }
});

type props_types = {
   file_clicked: Function,
   items: Array<{ id: string, name: string }>,
   actual_inx: Number
};
type state_types = {
   files: Array<string>,
   actual_inx: number
};

//@deprecated
export class TimeSelectionList extends Component<props_types, state_types> {
   constructor(props: props_types) {
      super(props);
      this.state = { files: [], actual_inx: -1 };
   }
   pickerupdate = (start_time: string, end_time: string) => {
      // start and end time in 24hour time
      console.log(`start time: ${start_time}, end time: ${end_time}`);
   };

   render() {
      console.log("time interval: " + this.props.timeInterval);
      var items = [];
      var five_min = 1000 * 60 * 5;
      var current_millis = new Date().getTime();
      var current_millis_rounded =
         current_millis - (current_millis % five_min) + five_min;
      const intervals = this.props.timeInterval * 1000 * 60;
      for (var i = 0; i < 100; i++)
         items.push(current_millis_rounded - intervals * i);

      return (
         <List>
            {items.map((item, inx) => (
               <ListItem
                  key={item.toString()}
                  onClick={() => {
                     this.props.setTimeWindow(item - intervals, item);
                     this.setState({ actual_inx: inx });
                  }}
                  style={{
                     backgroundColor:
                        inx == this.state.actual_inx ? "#00B1E1" : "#FFFFFF"
                  }}
                  button
               >
                  <ListItemText
                     primary={
                        moment(item).format("HH:mm:ss:SSS") +
                        "\n" +
                        moment(item - intervals).format("HH:mm:ss:SSS")
                     }
                  />
               </ListItem>
            ))}
         </List>
      );
   }
}
export default withStyles(styles)(TimeSelectionList);
