// @flow

import React from "react";
import PropTypes from "prop-types";
import { withStyles } from "@material-ui/core/styles";
import Typography from "@material-ui/core/Typography";
import Modal from "@material-ui/core/Modal";
import Button from "@material-ui/core/Button";
import { CopyToClipboard } from "react-copy-to-clipboard";
import { DOMPurify } from "dompurify";
function rand() {
  return Math.round(Math.random() * 20) - 10;
}

function getModalStyle() {
  const top = 50 + rand();
  const left = 50 + rand();

  return {
    top: `${top}%`,
    left: `${left}%`,
    transform: `translate(-${top}%, -${left}%)`
  };
}

const styles = theme => ({
  paper: {
    position: "absolute",
    backgroundColor: theme.palette.background.paper,
    width: 900,
    height: 700,
    overflow: "auto",
    whiteSpace: "pre-line",
    wordWrap: "break-word",
    boxShadow: theme.shadows[5],
    padding: theme.spacing.unit * 4
  }
});

class CopyModal extends React.Component<{
  classes: *,
  isOpen: boolean,
  onClose: () => *,
  text_to_copy: string,
  onCopy: string => void
}> {
  render() {
    const { classes } = this.props;

    return (
      <div>
        <Modal
          aria-labelledby="simple-modal-title"
          aria-describedby="simple-modal-description"
          open={this.props.isOpen}
          onClose={this.props.onClose}
        >
          <div style={getModalStyle()} className={classes.paper}>
            <Typography variant="title" id="modal-title">
              Request to copy
            </Typography>
            <pre id="to_copy">{this.props.text_to_copy}</pre>

            <CopyToClipboard text={this.props.text_to_copy}>
              <button>Copy</button>
            </CopyToClipboard>
            <SimpleModalWrapped />
          </div>
        </Modal>
      </div>
    );
  }
}

CopyModal.propTypes = {
  classes: PropTypes.object.isRequired
};

// We need an intermediary variable for handling the recursive nesting.
const SimpleModalWrapped = withStyles(styles)(CopyModal);

export default SimpleModalWrapped;
