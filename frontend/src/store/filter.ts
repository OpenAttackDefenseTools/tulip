import { createSlice, PayloadAction } from "@reduxjs/toolkit";

// Note: all off these states are immutable and can only be changed through overwrite
export interface TulipFilterState {
  filterTags: string[];
  includeTags: string[];
  excludeTags: string[];
  // can't use Map because immutable bs
  fuzzyHashes: string[];
  fuzzyHashIds: string[];
  includeFuzzyHashes: string[];
  excludeFuzzyHashes: string[];
  // startTick?: number;
  // endTick?: number;
  // service?: string;
  // textSearch?: string;
}

const initialState: TulipFilterState = {
  includeTags: [],
  excludeTags: [],
  filterTags: [],
  fuzzyHashes: [],
  fuzzyHashIds: [],
  includeFuzzyHashes: [],
  excludeFuzzyHashes: [],
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
    toggleFilterFuzzyHashes: (state, action: PayloadAction<string[]>) => {
      var fuzzyHashes = action.payload[0]
      var id = action.payload[1]
      var included = state.includeFuzzyHashes.includes(fuzzyHashes)
      var excluded = state.excludeFuzzyHashes.includes(fuzzyHashes)

      // If the fuzzyHashes hash is new cache it
      if(!state.fuzzyHashes.includes(fuzzyHashes)) {
        state.fuzzyHashes = [...state.fuzzyHashes, fuzzyHashes]
        state.fuzzyHashIds = [...state.fuzzyHashIds, id]
      }

      // If a user clicks a 'included' fuzzyHashes hash, the hash should be 'excluded' instead.
      if (included) {
        // Remove from included
        state.includeFuzzyHashes = state.includeFuzzyHashes.filter((t) => t !== fuzzyHashes);

        // Add to excluded
        state.excludeFuzzyHashes = [...state.excludeFuzzyHashes, fuzzyHashes]
      } else {
        // If the user clicks on an 'excluded' fuzzyHashes hash, the hash should be 'unset' from both include / exclude tags
        if (excluded) {
          // Remove from excluded
          state.excludeFuzzyHashes = state.excludeFuzzyHashes.filter((t) => t !== fuzzyHashes);
        } else {
          if (!included && !excluded) {
            // The tag was disabled, so it should be added to included now
            state.includeFuzzyHashes = [...state.includeFuzzyHashes, fuzzyHashes]
          }
        }
      }
    }
  },
});


export const { toggleFilterTag, toggleFilterFuzzyHashes } = filterSlice.actions;

export default filterSlice.reducer;
