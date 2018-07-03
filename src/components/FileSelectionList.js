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
import { fetchFiles } from "../data/fetcher";

const styles = theme => ({
   root: {
      width: "100%",
      backgroundColor: theme.palette.background.paper
   }
});

export class FileSelectionList extends Component<{
   file_clicked: Function,
   items: Array<{ id: string, name: string }>,
   actual_inx: Number
}> {
   constructor(props) {
      super(props);
      this.state = { files: [], actual_inx: -1 };
   }
   componentDidMount() {
      fetchFiles(files => {
         console.log("ok, ho i files: ");
         console.log(files);
         this.setState({ files: files });
         if (this.state.actual_inx == -1 && files.length > 0) {
            this.props.file_clicked(files[0]);
            this.setState({ actual_inx: 0 });
         }
      });
   }
   render() {
      var items = this.state.files || [];
      var actual_inx = this.state.actual_inx;
      return (
         <List>
            {items.map((item, inx) => (
               <ListItem
                  key={item}
                  onClick={() => {
                     this.props.file_clicked(item);
                     this.setState({ actual_inx: inx });
                  }}
                  style={{
                     backgroundColor: inx == actual_inx ? "#00B1E1" : "#FFFFFF"
                  }}
                  button
               >
                  <ListItemText primary={item} />
               </ListItem>
            ))}
         </List>
      );
   }
}
export default withStyles(styles)(FileSelectionList);
