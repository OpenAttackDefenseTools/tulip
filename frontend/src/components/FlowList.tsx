import { useSearchParams, Link } from "react-router-dom";
import { useEffect, useState } from "react";
import { useAtomValue } from "jotai";
import { api, Flow } from "../api";
import {
  fetchUrlAtom,
  SERVICE_FILTER_KEY,
  TEXT_FILTER_KEY,
  START_FILTER_KEY,
  END_FILTER_KEY,
} from "../App";

export function FlowList() {
  let [searchParams, setSearchParams] = useSearchParams();
  const services = useAtomValue(fetchUrlAtom);
  const [flowList, setFlowList] = useState<Flow[]>([]);

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
      });
      setFlowList(data);
    };
    fetchData().catch(console.error);
  }, [service, text_filter, from_filter, to_filter]);

  return (
    <div>
      <ul>
        {flowList.map((flow) => (
          <Link to={`/flow/${flow._id.$oid}?${searchParams}`}>
            <FlowListEntry key={flow._id.$oid} flow={flow} />
          </Link>
        ))}
      </ul>
    </div>
  );
}
function FlowListEntry({ flow }: { flow: Flow }) {
  return (
    <li>
      <div>
        <div>
          <p>
            {flow.src_ip}:{flow.src_port} - {flow.dst_ip}:{flow.dst_port}
          </p>
        </div>
      </div>
    </li>
  );
}
