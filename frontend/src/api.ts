import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

import { API_BASE_PATH } from "./const";
import {
  Service,
  FullFlow,
  Signature,
  TickInfo,
  Flow,
  FlowsQuery,
} from "./types";

export const tulipApi = createApi({
  baseQuery: fetchBaseQuery({ baseUrl: API_BASE_PATH }),
  endpoints: (builder) => ({
    getServices: builder.query<Service[], void>({
      query: () => "/services",
    }),
    getFlow: builder.query<FullFlow, string>({
      query: (id) => `/flow/${id}`,
    }),
    getFlows: builder.query<Flow[], FlowsQuery>({
      query: (query) => ({
        url: `/query`,
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        // TODO: fix the below tags mutation (make backend handle empty tags!)
        // Diederik gives you a beer once this has been fixed
        body: JSON.stringify({
          ...query,
          includeTags: query.includeTags.length > 0 ? query.includeTags : undefined,
          excludeTags: query.excludeTags.length > 0 ? query.excludeTags : undefined,
        }),
      }),
    }),
    getTags: builder.query<string[], void>({
      query: () => `/tags`,
    }),
    getTickInfo: builder.query<TickInfo, void>({
      query: () => `/tick_info`,
    }),
    getSignature: builder.query<Signature[], number>({
      query: (id) => `/signature/${id}`,
    }),
    toPwnTools: builder.query<string, string>({
      query: (id) => ({ url: `/to_pwn/${id}`, responseHandler: "text" }),
    }),
    toSinglePythonRequest: builder.query<
      string,
      { body: string; id: string; tokenize: boolean }
    >({
      query: ({ body, id, tokenize }) => ({
        url: `/to_single_python_request?tokenize=${
          tokenize ? "1" : "0"
        }&id=${id}`,
        method: "POST",
        responseHandler: "text",
        headers: {
          "Content-Type": "text/plain;charset=UTF-8",
        },
        body,
      }),
    }),
    toFullPythonRequest: builder.query<string, string>({
      query: (id) => ({
        url: `/to_python_request/${id}`,
        responseHandler: "text",
      }),
    }),
    starFlow: builder.mutation<unknown, { id: string; star: boolean }>({
      query: ({ id, star }) => `/star/${id}/${star ? "1" : "0"}`,
      // TODO: optimistic cache update

      // async onQueryStarted({ id, star }, { dispatch, queryFulfilled }) {
      //   // `updateQueryData` requires the endpoint name and cache key arguments,
      //   // so it knows which piece of cache state to update
      //   const patchResult = dispatch(
      //     tulipApi.util.updateQueryData("getFlows", undefined, (flows) => {
      //       // The `flows` is Immer-wrapped and can be "mutated" like in createSlice
      //       const flow = flows.find((flow) => flow._id.$oid === id);
      //       if (flow) {
      //         if (star) {
      //           flow.tags.push("starred");
      //         } else {
      //           flow.tags = flow.tags.filter((tag) => tag != "starred");
      //         }
      //       }
      //     })
      //   );
      //   try {
      //     await queryFulfilled;
      //   } catch {
      //     patchResult.undo();
      //   }
      // },
    }),
  }),
});

export const {
  useGetServicesQuery,
  useGetFlowQuery,
  useGetFlowsQuery,
  useLazyGetFlowsQuery,
  useGetTagsQuery,
  useGetSignatureQuery,
  useGetTickInfoQuery,
  useLazyToPwnToolsQuery,
  useLazyToFullPythonRequestQuery,
  useToSinglePythonRequestQuery,
  useStarFlowMutation,
} = tulipApi;
