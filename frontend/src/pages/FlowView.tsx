import { useSearchParams, Link, useParams } from "react-router-dom";
import React, { useState } from "react";
import { FlowData, FullFlow } from "../types";

import {
  ArrowCircleLeftIcon,
  ArrowCircleRightIcon,
} from "@heroicons/react/solid";
import { format } from "date-fns";

import { hexy } from "hexy";
import { useCopy } from "../hooks/useCopy";
import { RadioGroup, RadioGroupProps } from "../components/RadioGroup";
import {
  useGetFlowQuery,
  useLazyToFullPythonRequestQuery,
  useLazyToPwnToolsQuery,
  useToSinglePythonRequestQuery,
} from "../api";
import { API_BASE_PATH } from "../const";

const SECONDARY_NAVBAR_HEIGHT = 50;

function CopyButton({ copyText }: { copyText?: string }) {
  const { statusText, copy, copyState } = useCopy({
    getText: async () => copyText ?? "",
  });
  return (
    <>
      {copyText && (
        <button
          className="p-2 text-sm text-blue-500"
          onClick={copy}
          disabled={!copyText}
        >
          {statusText}
        </button>
      )}
    </>
  );
}

function FlowContainer({
  copyText,
  children,
}: {
  copyText?: string;
  children: React.ReactNode;
}) {
  return (
    <div className=" pb-5 flex flex-col">
      <div className="ml-auto">
        <CopyButton copyText={copyText}></CopyButton>
      </div>
      <pre className="p-5 overflow-auto">{children}</pre>
    </div>
  );
}

function HexFlow({ flow }: { flow: FlowData }) {
  const hex = hexy(atob(flow.b64), { format: "twos" });
  return <FlowContainer copyText={hex}>{hex}</FlowContainer>;
}

function TextFlow({ flow }: { flow: FlowData }) {
  return <FlowContainer copyText={flow.data}>{flow.data}</FlowContainer>;
}

function WebFlow({ flow }: { flow: FlowData }) {
  const data = flow.data;
  const [header, ...rest] = data.split("\r\n\r\n");
  const http_content = rest.join("\r\n\r\n");

  const Hack = "iframe" as any;
  return (
    <FlowContainer>
      <pre>{header}</pre>
      <div className="border border-gray-200 rounded-lg">
        <Hack
          srcDoc={http_content}
          sandbox=""
          height={300}
          csp="default-src none" // there is a warning here but it actually works, not supported in firefox though :(
        ></Hack>
      </div>
    </FlowContainer>
  );
}

function PythonRequestFlow({
  full_flow,
  flow,
}: {
  full_flow: FullFlow;
  flow: FlowData;
}) {
  const { data } = useToSinglePythonRequestQuery({
    body: flow.b64,
    id: full_flow._id.$oid,
    tokenize: true,
  });

  return <FlowContainer copyText={data}>{data}</FlowContainer>;
}

interface FlowProps {
  full_flow: FullFlow;
  flow: FlowData;
  delta_time: number;
}

function detectType(flow: FlowData) {
  const firstLine = flow.data.split("\n")[0];
  if (firstLine.includes("HTTP")) {
    return "Plain";
  }

  return "Plain";
}

function Flow({ full_flow, flow, delta_time }: FlowProps) {
  const formatted_time = format(new Date(flow.time), "HH:mm:ss:SSS");
  const displayOptions = ["Plain", "Hex", "Web", "PythonRequest"];

  // Basic type detection, currently unused
  const [displayOption, setDisplayOption] = useState(detectType(flow));

  return (
    <div className=" text-mono">
      <div
        className="sticky shadow-md bg-white overflow-auto py-1 border-y"
        style={{ top: SECONDARY_NAVBAR_HEIGHT }}
      >
        <div className="flex items-center h-6">
          <div className="w-8 px-2">
            {flow.from === "s" ? (
              <ArrowCircleLeftIcon className="fill-green-700" />
            ) : (
              <ArrowCircleRightIcon className="fill-red-700" />
            )}
          </div>
          <div style={{ width: 200 }}>
            {formatted_time}
            <span className="text-gray-400 pl-3">{delta_time}ms</span>
          </div>
          <button
            className="bg-gray-200 py-1 px-2 rounded-md text-sm"
            onClick={async () => {
              window.open(
                "https://gchq.github.io/CyberChef/#input=" +
                  encodeURIComponent(flow.b64)
              );
            }}
          >
            Open in cyberchef
          </button>
          <RadioGroup
            options={displayOptions}
            value={displayOption}
            onChange={setDisplayOption}
            className="flex gap-2 text-gray-800 text-sm mr-4 ml-auto"
          />
        </div>
      </div>
      <div
        className={
          flow.from === "s"
            ? "border-l-8 border-green-300"
            : "border-l-8 border-red-300"
        }
      >
        {displayOption === "Hex" && <HexFlow flow={flow}></HexFlow>}
        {displayOption === "Plain" && <TextFlow flow={flow}></TextFlow>}
        {displayOption === "Web" && <WebFlow flow={flow}></WebFlow>}
        {displayOption === "PythonRequest" && (
          <PythonRequestFlow
            flow={flow}
            full_flow={full_flow}
          ></PythonRequestFlow>
        )}
      </div>
    </div>
  );
}

// Helper function to format the IP for display. If the IP contains ":",
// assume it is an ipv6 address and surround it in square brackets
function formatIP(ip: string) {
  return ip.includes(":") ? `[${ip}]` : ip;
}

function FlowOverview({ flow }: { flow: FullFlow }) {
  return (
    <div>
      {flow.signatures?.length > 0 ? (
        <div className="bg-blue-200 p-2">
          <div className="font-extrabold">Suricata</div>
          <div className="pl-2">
            {flow.signatures.map((sig) => {
              return (
                <div className="py-1">
                  <div className="flex">
                    <div>Message: </div>
                    <div className="font-bold">{sig.msg}</div>
                  </div>
                  <div className="flex">
                    <div>Rule ID: </div>
                    <div className="font-bold">{sig.id}</div>
                  </div>
                  <div className="flex">
                    <div>Action taken: </div>
                    <div
                      className={
                        sig.action === "blocked"
                          ? "font-bold text-red-800"
                          : "font-bold text-green-800"
                      }
                    >
                      {sig.action}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      ) : undefined}
      <div className="bg-yellow-200 p-2">
        <div className="font-extrabold">Meta</div>
        <div className="pl-2">
          <div>Source: </div>
          <div className="font-bold">
            <a href={`${API_BASE_PATH}/download/?file=${flow.filename}`}>
              {flow.filename}
            </a>
          </div>
          <div></div>
          <div>Tags: </div>
          <div className="font-bold">[{flow.tags.join(", ")}]</div>
          <div></div>
          <div>Source - Target: </div>
          <div className="flex items-center gap-1">
            <div>
              {" "}
              <span>{formatIP(flow.src_ip)}</span>:
              <span className="font-bold">{flow.src_port}</span>
            </div>
            <div>-</div>
            <div>
              <span>{formatIP(flow.dst_ip)}</span>:
              <span className="font-bold">{flow.dst_port}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export function FlowView() {
  let [searchParams] = useSearchParams();
  const params = useParams();

  const id = params.id;

  const { data: flow } = useGetFlowQuery(id!, { skip: id === undefined });

  const [triggerPwnToolsQuery] = useLazyToPwnToolsQuery();
  const [triggerFullPythonRequestQuery] = useLazyToFullPythonRequestQuery();

  async function copyAsPwn() {
    if (flow?._id.$oid) {
      const { data } = await triggerPwnToolsQuery(flow?._id.$oid);
      console.log(data);
      return data || "";
    }
    return "";
  }

  const { statusText: pwnCopyStatusText, copy: copyPwn } = useCopy({
    getText: copyAsPwn,
    copyStateToText: {
      copied: "Copied",
      default: "Copy as pwntools",
      failed: "Failed",
      copying: "Generating payload",
    },
  });

  async function copyAsRequests() {
    if (flow?._id.$oid) {
      const { data } = await triggerFullPythonRequestQuery(flow?._id.$oid);
      return data || "";
    }
    return "";
  }

  const { statusText: requestsCopyStatusText, copy: copyRequests } = useCopy({
    getText: copyAsRequests,
    copyStateToText: {
      copied: "Copied",
      default: "Copy as requests",
      failed: "Failed",
      copying: "Generating payload",
    },
  });

  if (flow === undefined) {
    return <div>Loading...</div>;
  }

  return (
    <div>
      <div
        className="sticky shadow-md top-0 bg-white overflow-auto border-b border-b-gray-200 flex"
        style={{ height: SECONDARY_NAVBAR_HEIGHT, zIndex: 100 }}
      >
        <div className="flex align-middle p-2 gap-3 ml-auto">
          <button
            className="bg-gray-700 text-white px-2 text-sm rounded-md"
            onClick={copyPwn}
          >
            {pwnCopyStatusText}
          </button>

          <button
            className="bg-gray-700 text-white px-2 text-sm rounded-md"
            onClick={copyRequests}
          >
            {requestsCopyStatusText}
          </button>
        </div>
      </div>
      {flow?.parent_id?.$oid != "000000000000000000000000" ? (
        <Link
          to={`/flow/${flow.parent_id.$oid}?${searchParams}`}
          key={flow.parent_id.$oid}
          className="focus-visible:rounded-md"
        >
          Parent
        </Link>
      ) : undefined}

      {flow ? <FlowOverview flow={flow}></FlowOverview> : undefined}
      {flow?.flow.map((flow_data, i, a) => {
        const delta_time = a[i].time - (a[i - 1]?.time ?? a[i].time);
        return (
          <Flow
            flow={flow_data}
            delta_time={delta_time}
            full_flow={flow}
            key={flow._id.$oid + " " + i}
          ></Flow>
        );
      })}

      {flow?.child_id?.$oid != "000000000000000000000000" ? (
        <Link
          to={`/flow/${flow.child_id.$oid}?${searchParams}`}
          key={flow.child_id.$oid}
          className="focus-visible:rounded-md"
        >
          Child
        </Link>
      ) : undefined}
    </div>
  );
}
