import { createSlice, PayloadAction } from "@reduxjs/toolkit";

export interface TulipFilterState {
  includeTags: string[];
  excludeTags: string[];
  // startTick?: number;
  // endTick?: number;
  // service?: string;
  // textSearch?: string;
}

const initialState: TulipFilterState = {
  includeTags: [],
  excludeTags: [],
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
  },
});

export const { toggleFilterTag } = filterSlice.actions;

export default filterSlice.reducer;
