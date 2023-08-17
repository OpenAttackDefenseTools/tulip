import { createSlice, PayloadAction } from "@reduxjs/toolkit";

export interface TulipFilterState {
  filterTags: string[];
  filterFlags: string[];
  filterFlagids: string[];
  // startTick?: number;
  // endTick?: number;
  // service?: string;
  // textSearch?: string;
}

const initialState: TulipFilterState = {
  filterTags: [],
  filterFlags: [],
  filterFlagids: [],
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
      state.filterTags = state.filterTags.includes(action.payload)
        ? state.filterTags.filter((t) => t !== action.payload)
        : [...state.filterTags, action.payload];
    },
    toggleFilterFlags: (state, action: PayloadAction<string>) => {
      state.filterFlags = state.filterFlags.includes(action.payload)
          ? state.filterFlags.filter((t) => t !== action.payload)
          : [...state.filterFlags, action.payload];
    },
    toggleFilterFlagids: (state, action: PayloadAction<string>) => {
      state.filterFlagids = state.filterFlagids.includes(action.payload)
          ? state.filterFlagids.filter((t) => t !== action.payload)
          : [...state.filterFlagids, action.payload];
    },
  },
});

export const { toggleFilterTag } = filterSlice.actions;

export default filterSlice.reducer;
