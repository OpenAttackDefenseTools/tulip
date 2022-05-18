import { useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import { api, FlowData, FullFlow } from "../api";
import { showHexAtom } from "../components/Header";
import { useAtom } from "jotai";

import {
  ArrowCircleLeftIcon,
  ArrowCircleRightIcon,
} from "@heroicons/react/solid";
import { format } from "date-fns";

function HexFlow({ flow }: { flow: FlowData }) {
  return <pre>{flow.hex}</pre>;
}

function TextFlow({ flow }: { flow: FlowData }) {
  return <pre className="p-5">{flow.data}</pre>;
}

interface FlowProps {
  flow: FlowData;
  useHex: boolean;
  delta_time: number;
}

function Flow({ flow, useHex, delta_time }: FlowProps) {
  const formatted_time = format(new Date(flow.time), "HH:mm:ss:SSS");

  return (
    <>
      <div
        className="sticky shadow-md bg-white overflow-scroll py-1 border-y"
        style={{ top: 30 }}
      >
        <div className="flex items-center h-6">
          <div className="w-8 px-2">
            {flow.from === "s" ? (
              <ArrowCircleLeftIcon className="fill-green-700" />
            ) : (
              <ArrowCircleRightIcon className="fill-red-700" />
            )}
          </div>
          <div>
            {formatted_time}
            <span className="text-gray-400 pl-3">{delta_time}ms</span>
          </div>
        </div>
      </div>
      <div
        className={
          flow.from === "s"
            ? "border-l-8 border-green-300"
            : "border-l-8 border-red-300"
        }
      >
        {useHex && <HexFlow flow={flow}></HexFlow>}
        {!useHex && <TextFlow flow={flow}></TextFlow>}
      </div>
    </>
  );
}

function Header() {}

export function FlowView() {
  const params = useParams();
  const [flow, setFlow] = useState<FullFlow>();
  const [useHex] = useAtom(showHexAtom);

  const id = params.id;

  useEffect(() => {
    const fetchData = async () => {
      if (id === undefined) {
        return;
      }
      const data = await api.getFlow(id);
      setFlow(data);
    };
    fetchData().catch(console.error);
  }, [id]);

  console.log(flow);

  return (
    <div>
      <div
        className="sticky shadow-md top-0 bg-white overflow-scroll border-b border-b-gray-200"
        style={{ height: 30 }}
      >
        {}
      </div>
      {flow?.flow.map((flow_data, i, a) => {
        const delta_time = a[i].time - (a[i - 1]?.time ?? a[i].time);
        return (
          <Flow flow={flow_data} useHex={useHex} delta_time={delta_time}></Flow>
        );
      })}
    </div>
  );
}
