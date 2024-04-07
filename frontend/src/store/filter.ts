import { createSlice, PayloadAction } from "@reduxjs/toolkit";

// Note: all off these states are immutable and can only be changed through overwrite
export interface TulipFilterState {
  filterTags: string[];
  includeTags: string[];
  excludeTags: string[];
  // can't use Map because immutable bs
  ssdeeps: string[];
  ssdeep_ids: string[];
  includeSsdeep: string[];
  excludeSsdeep: string[];
  // startTick?: number;
  // endTick?: number;
  // service?: string;
  // textSearch?: string;
}

const initialState: TulipFilterState = {
  includeTags: [],
  excludeTags: [],
  filterTags: [],
  ssdeeps: [],
  ssdeep_ids: [],
  includeSsdeep: [],
  excludeSsdeep: [],
};

export const filterSlice = createSlice({
  name: "filter",
  initialState,
  reducers: {
    // updateStartTick: (state, action: PayloadAction<number>) => {
    //   state.startTick = action.payload;
    // },
    // updateEndTick: (state, action: PayloadAction<number>) => {
    //   state.endTick = action.payload;
    // },
    toggleFilterTag: (state, action: PayloadAction<string>) => {
      var included = state.includeTags.includes(action.payload)
      var excluded = state.excludeTags.includes(action.payload)

      // If a user clicks a 'included' tag, the tag should be 'excluded' instead.
      if (included) {
        // Remove from included
        state.includeTags = state.includeTags.filter((t) => t !== action.payload);

        // Add to excluded
        state.excludeTags = [...state.excludeTags, action.payload]
      } else {
        // If the user clicks on an 'excluded' tag, the tag should be 'unset' from both include / exclude tags
        if (excluded) {
          // Remove from excluded
          state.excludeTags = state.excludeTags.filter((t) => t !== action.payload);
        } else {
          if (!included && !excluded) {
            // The tag was disabled, so it should be added to included now
            state.includeTags = [...state.includeTags, action.payload]
          }
        }
      }
    },
    toggleFilterSsdeep: (state, action: PayloadAction<string[]>) => {
      var ssdeep = action.payload[0]
      var id = action.payload[1]
      var included = state.includeSsdeep.includes(ssdeep)
      var excluded = state.excludeSsdeep.includes(ssdeep)

      // If the ssdeep hash is new cache it
      if(!state.ssdeeps.includes(ssdeep)) {
        state.ssdeeps = [...state.ssdeeps, ssdeep]
        state.ssdeep_ids = [...state.ssdeep_ids, id]
      }

      // If a user clicks a 'included' ssdeep hash, the hash should be 'excluded' instead.
      if (included) {
        // Remove from included
        state.includeSsdeep = state.includeSsdeep.filter((t) => t !== ssdeep);

        // Add to excluded
        state.excludeSsdeep = [...state.excludeSsdeep, ssdeep]
      } else {
        // If the user clicks on an 'excluded' ssdeep hash, the hash should be 'unset' from both include / exclude tags
        if (excluded) {
          // Remove from excluded
          state.excludeSsdeep = state.excludeSsdeep.filter((t) => t !== ssdeep);
        } else {
          if (!included && !excluded) {
            // The tag was disabled, so it should be added to included now
            state.includeSsdeep = [...state.includeSsdeep, ssdeep]
          }
        }
      }
    }
  },
});


export const { toggleFilterTag, toggleFilterSsdeep } = filterSlice.actions;

export default filterSlice.reducer;
