import {
  useSearchParams,
  Link,
  useParams,
  useNavigate,
} from "react-router-dom";
import { useCallback, useEffect, useState } from "react";
import { useAtom, useAtomValue } from "jotai";
import { api, Flow } from "../api";
import {
  fetchUrlAtom,
  SERVICE_FILTER_KEY,
  TEXT_FILTER_KEY,
  START_FILTER_KEY,
  END_FILTER_KEY,
} from "../App";

import { HeartIcon } from "@heroicons/react/solid";
import { HeartIcon as EmptyHeartIcon } from "@heroicons/react/outline";

import classes from "./FlowList.module.css";
import { format } from "date-fns";
import { atomWithStorage } from "jotai/utils";

const onlyStarred = atomWithStorage("onlyStarred", false);

export function FlowList() {
  let [searchParams] = useSearchParams();
  let params = useParams();
  const services = useAtomValue(fetchUrlAtom);
  const [flowList, setFlowList] = useState<Flow[]>([]);
  const [starred, setStarred] = useAtom(onlyStarred);

  const service_name = searchParams.get(SERVICE_FILTER_KEY) ?? "";
  const service = services.find((s) => s.name == service_name);

  const text_filter = searchParams.get(TEXT_FILTER_KEY) ?? undefined;
  const from_filter = searchParams.get(START_FILTER_KEY) ?? undefined;
  const to_filter = searchParams.get(END_FILTER_KEY) ?? undefined;

  useEffect(() => {
    const fetchData = async () => {
      const data = await api.getFlows({
        "flow.data": text_filter,
        dst_ip: service?.ip,
        dst_port: service?.port,
        from_time: from_filter,
        to_time: to_filter,
        starred: starred ? 1 : undefined,
      });
      setFlowList(data);
    };
    fetchData().catch(console.error);
  }, [service, text_filter, from_filter, to_filter, starred]);

  const onHeartHandler = useCallback(
    async (flow: Flow) => {
      const star_res = await api.starFlow(flow._id.$oid, !flow.starred);
      // todo error handling star res
      const data = await api.getFlows({
        "flow.data": text_filter,
        dst_ip: service?.ip,
        dst_port: service?.port,
        from_time: from_filter,
        to_time: to_filter,
        starred: starred ? 1 : undefined,
      });
      setFlowList(data);
    },
    [service, text_filter, from_filter, to_filter, starred]
  );

  return (
    <div className="">
      <div className="sticky top-0 bg-white p-2 border-b-gray-300 border-b shadow-md">
        <input
          type="checkbox"
          className="mr-2"
          checked={starred}
          onChange={() => {
            setStarred(!starred);
          }}
        />
        <label htmlFor="">Show only starred</label>
      </div>
      <ul className={classes.list_container}>
        {flowList.map((flow) => (
          <Link to={`/flow/${flow._id.$oid}?${searchParams}`}>
            <FlowListEntry
              key={flow._id.$oid}
              flow={flow}
              isActive={flow._id.$oid === params.id}
              onHeartClick={onHeartHandler}
            />
          </Link>
        ))}
      </ul>
    </div>
  );
}

interface FlowListEntryProps {
  flow: Flow;
  isActive: boolean;
  onHeartClick: (flow: Flow) => void;
}

function FlowListEntry({ flow, isActive, onHeartClick }: FlowListEntryProps) {
  const formatted_time = format(new Date(flow.time), "HH:mm:ss SSS");
  return (
    <li className={isActive ? classes.active : undefined}>
      <div className="flex">
        <div
          className="w-5 ml-2 mr-4 self-center"
          onClick={() => {
            onHeartClick(flow);
          }}
        >
          {flow.starred ? (
            <HeartIcon className="text-red-500" />
          ) : (
            <EmptyHeartIcon />
          )}
        </div>
        <div className="flex-1">
          <div className="flex justify-between">
            <div>
              {" "}
              <span>{flow.src_ip}</span>:
              <span className="font-bold">{flow.src_port}</span>
            </div>
            <div>
              <span>{flow.dst_ip}</span>:
              <span className="font-bold">{flow.dst_port}</span>
            </div>
          </div>
          <div className="flex justify-between text-gray-500">
            <div>{formatted_time}</div>
            <div>{flow.duration}ms</div>
          </div>
        </div>
      </div>
    </li>
  );
}
