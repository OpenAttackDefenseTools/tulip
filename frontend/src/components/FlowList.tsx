import {
  useSearchParams,
  Link,
  useParams,
  useNavigate,
} from "react-router-dom";
import { useState, useRef, useEffect } from "react";
import { useHotkeys } from 'react-hotkeys-hook';
import { FetchBaseQueryError } from '@reduxjs/toolkit/query'
import { Flow } from "../types";
import {
  SERVICE_FILTER_KEY,
  TEXT_FILTER_KEY,
  START_FILTER_KEY,
  END_FILTER_KEY,
  FLOW_LIST_REFETCH_INTERVAL_MS,
} from "../const";
import { useAppSelector, useAppDispatch } from "../store";
import { toggleFilterTag, toggleTagIntersectMode } from "../store/filter";

import { HeartIcon, FilterIcon, LinkIcon } from "@heroicons/react/solid";
import { HeartIcon as EmptyHeartIcon } from "@heroicons/react/outline";

import classes from "./FlowList.module.css";
import { format } from "date-fns";
import useDebounce from "../hooks/useDebounce";
import { Virtuoso, VirtuosoHandle } from "react-virtuoso";
import classNames from "classnames";
import { Tag } from "./Tag";
import {
  useGetFlowsQuery,
  useGetServicesQuery,
  useGetTagsQuery,
  useStarFlowMutation,
} from "../api";

export function FlowList() {
  let [searchParams] = useSearchParams();
  let params = useParams();

  // we add a local variable to prevent racing with the browser location API
  let openedFlowID = params.id

  const { data: availableTags } = useGetTagsQuery();
  const { data: services } = useGetServicesQuery();

  const filterFlags = useAppSelector((state) => state.filter.filterFlags);
  const filterFlagids = useAppSelector((state) => state.filter.filterFlagids);
  const includeTags = useAppSelector((state) => state.filter.includeTags);
  const excludeTags = useAppSelector((state) => state.filter.excludeTags);
  const tagIntersectionMode = useAppSelector((state) => state.filter.tagIntersectionMode);

  const dispatch = useAppDispatch();

  const [starFlow] = useStarFlowMutation();

  const [flowIndex, setFlowIndex] = useState<number>(0);

  const virtuoso = useRef<VirtuosoHandle>(null);

  const service_name = searchParams.get(SERVICE_FILTER_KEY) ?? "";
  const service = services?.find((s) => s.name == service_name);

  const text_filter = searchParams.get(TEXT_FILTER_KEY) ?? undefined;
  const from_filter = searchParams.get(START_FILTER_KEY) ?? undefined;
  const to_filter = searchParams.get(END_FILTER_KEY) ?? undefined;

  const debounced_text_filter = useDebounce(text_filter, 300);

  const {
    data: flowData, error: flowQueryError,
    isLoading, isFetching, refetch,
    startedTimeStamp, fulfilledTimeStamp,
  } = useGetFlowsQuery(
    {
      regex_insensitive: debounced_text_filter,
      ip_dst: service?.ip,
      port_dst: service?.port,
      time_from: from_filter ? new Date(parseInt(from_filter)).toISOString() : undefined,
      time_to: to_filter ? new Date(parseInt(to_filter)).toISOString() : undefined,
      tags_include: includeTags,
      tags_exclude: excludeTags,
      tag_intersection_mode: tagIntersectionMode,
      flags: filterFlags,
      flagids: filterFlagids,
    },
    {
      refetchOnMountOrArgChange: true,
      pollingInterval: FLOW_LIST_REFETCH_INTERVAL_MS,
    }
  );

  interface FlowQueryError { error: string }
  const isFetchBaseQueryError = (error: unknown): error is FetchBaseQueryError =>
    typeof error === 'object' && error != null && 'status' in error
  const isFlowQueryError = (error: unknown): error is FlowQueryError =>
    typeof error === 'object' && error != null && 'error' in error
  const flowQueryErrorMessage = isFetchBaseQueryError(flowQueryError)
    && isFlowQueryError(flowQueryError.data)
    ? flowQueryError.data.error : null;

  let searchMessage = null;
  if(isFetching)
    searchMessage = "Searching...";
  else if(flowQueryErrorMessage)
    searchMessage = `Error: ${flowQueryErrorMessage}`;
  else if(startedTimeStamp && fulfilledTimeStamp)
    searchMessage = `Search took ${fulfilledTimeStamp - startedTimeStamp}ms`

  // TODO: fix the below transformation - move it to server
  // Diederik gives you a beer once it has been fixed
  const transformedFlowData = flowData?.map((flow) => ({
    ...flow,
    service_tag:
      services?.find((s) => s.ip === flow.dst_ip && s.port === flow.dst_port)
        ?.name ?? "unknown",
  }));

  const onHeartHandler = async (flow: Flow) => {
    await starFlow({ id: flow.id, star: !flow.tags.includes("starred") });
  };

  const navigate = useNavigate();

  useEffect(() => {
      virtuoso?.current?.scrollIntoView({
        index: flowIndex,
        behavior: 'auto',
        done: () => {
          if (transformedFlowData && transformedFlowData[flowIndex ?? 0]) {
            let idAtIndex = transformedFlowData[flowIndex ?? 0].id;
            // if the current flow ID at the index indeed did change (ie because of keyboard navigation), we need to update the URL as well as local ID
            if (idAtIndex !== openedFlowID) {
              navigate(`/flow/${idAtIndex}?${searchParams}`)
              openedFlowID = idAtIndex
            }
          }
        },
      })
    },
    [flowIndex]
  )

  // TODO: there must be a better way to do this
  // this gets called on every refetch, we dont want to iterate all flows on every refetch
  // so because performance, we hack this by checking if the transformedFlowData length changed
  const [transformedFlowDataLength, setTransformedFlowDataLength] = useState<number>(0);
  useEffect(
    () => {
      if (transformedFlowData && transformedFlowDataLength != transformedFlowData?.length) {
        setTransformedFlowDataLength(transformedFlowData?.length)

        for (let i = 0; i < transformedFlowData?.length; i++) {
          if (transformedFlowData[i].id === openedFlowID) {
            if (i !== flowIndex) {
              setFlowIndex(i)
            }
            return
          }
        }
        setFlowIndex(0)
      }
    },
    [transformedFlowData]
  )

  useHotkeys('j', () => setFlowIndex(fi => Math.min((transformedFlowData?.length ?? 1)-1, fi + 1)), [transformedFlowData?.length]);
  useHotkeys('k', () => setFlowIndex(fi => Math.max(0, fi - 1)));
  useHotkeys('i', () => {
    setShowFilters(true)
    if ((availableTags ?? []).includes("flag-in")) {
      dispatch(toggleFilterTag("flag-in"))
    }
  }, [availableTags]);
  useHotkeys('o', () => {
    setShowFilters(true)
    if ((availableTags ?? []).includes("flag-out")) {
      dispatch(toggleFilterTag("flag-out"))
    }
  }, [availableTags]);
  useHotkeys('r', () => refetch());

  const [showFilters, setShowFilters] = useState(false);

  return (
    <div className="flex flex-col h-full">
      <div className="bg-white border-b-gray-300 border-b shadow-md flex flex-col">
        <div className="p-2 flex" style={{ height: 50 }}>
          <button
            className="flex gap-1 items-center text-sm"
            onClick={() => setShowFilters(!showFilters)}
          >
            {<FilterIcon height={20} className="text-gray-400"></FilterIcon>}
            {showFilters ? "Close" : "Open"} filters
          </button>
          {/* Maybe we want to use a search button instead of live search */}
          {false && (
            <button className="ml-auto items-center bg-blue-600 text-white px-2 rounded-md text-sm">
              Search
            </button>
          )}
        </div>
        {showFilters && (
          <div className="border-t-gray-300 border-t p-2">
            <div className="flex">
              <p className="text-sm font-bold text-gray-600 pb-2">
                Intersection filter
              </p>
              <button
                className="w-24 h-5 bg-blue-100 text-sm rounded-md ml-auto"
                onClick={() => dispatch(toggleTagIntersectMode())}
              >
                Mode:&nbsp;{tagIntersectionMode}
              </button>
            </div>
            <div className="flex gap-2 flex-wrap">
              {(availableTags ?? []).map((tag) => (
                <Tag
                  key={tag}
                  tag={tag}
                  disabled={!includeTags.includes(tag)}
                  excluded={excludeTags.includes(tag)}
                  onClick={() => dispatch(toggleFilterTag(tag))}
                ></Tag>
              ))}
            </div>
          </div>
        )}
      </div>
      <div></div>
      { searchMessage && <div>{searchMessage}</div> }
      <Virtuoso
        className={classNames({
          "flex-1": true,
          [classes.list_container]: true,
          "sidebar-loading": isLoading,
        })}
        data={transformedFlowData}
        ref={virtuoso}
        initialTopMostItemIndex={flowIndex}
        itemContent={(index, flow) => (
          <Link
            to={`/flow/${flow.id}?${searchParams}`}
            onClick={() => setFlowIndex(index)}
            key={flow.id}
            className="focus-visible:rounded-md"
            //style={{ paddingTop: '1em' }}
          >
            <FlowListEntry
              key={flow.id}
              flow={flow}
              isActive={flow.id === openedFlowID}
              onHeartClick={onHeartHandler}
            />
          </Link>
        )}
      />
    </div>
  );
}

interface FlowListEntryProps {
  flow: Flow;
  isActive: boolean;
  onHeartClick: (flow: Flow) => void;
}

function FlowListEntry({ flow, isActive, onHeartClick }: FlowListEntryProps) {
  const formatted_time_h_m_s = format(new Date(flow.time), "HH:mm:ss");
  const formatted_time_ms = format(new Date(flow.time), ".SSS");

  const [isStarred, setStarred] = useState(flow.tags.includes("starred"));

  // Filter tag list for tags that are handled specially
  const filtered_tag_list = flow.tags.filter((t) => t != "starred");

  const duration =
    flow.duration > 10000 ? (
      <div className="text-red-500">&gt;10s</div>
    ) : (
      <div>{flow.duration}ms</div>
    );
  return (
    <li
      className={classNames({
        [classes.active]: isActive,
      })}
    >
      <div className="flex">
        <div
          className="w-5 ml-1 mr-1 self-center shrink-0"
          onClick={() => {
            setStarred(!isStarred);
            onHeartClick(flow);
          }}
        >
          {isStarred ? (
            <HeartIcon className="text-red-500" />
          ) : (
            <EmptyHeartIcon />
          )}
        </div>

        <div className="w-5 mr-2 self-center shrink-0">
          {flow.child_id != null || flow.parent_id != null ? (
            <LinkIcon className="text-blue-500" />
          ) : undefined}
        </div>
        <div className="flex-1 shrink">
          <div className="flex">
            <div className="shrink-0">
              <span className="text-gray-700 font-bold overflow-ellipsis overflow-hidden ">
                {flow.service_tag}
              </span>
              <span className="text-gray-500">:{flow.dst_port}</span>
            </div>

            <div className="ml-2">
              <span className="text-gray-500">{formatted_time_h_m_s}</span>
              <span className="text-gray-300">{formatted_time_ms}</span>
            </div>
            <div className="text-gray-500 ml-auto">{duration}</div>
          </div>
          <div className="flex gap-2 flex-wrap">
            {filtered_tag_list.map((tag) => (
              <Tag key={tag} tag={tag}></Tag>
            ))}
          </div>
        </div>
      </div>
    </li>
  );
}

export { FlowListEntry };
