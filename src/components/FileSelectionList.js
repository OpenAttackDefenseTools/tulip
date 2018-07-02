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
