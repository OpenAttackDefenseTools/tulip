import { useSearchParams, Link, useParams } from "react-router-dom";
import { useState } from "react";
import { Buffer } from "buffer";

import { FullFlow } from "../types";

import ReactDiffViewer from "react-diff-viewer";
import { RadioGroup } from "../components/RadioGroup";

import { hexy } from "hexy";

import { FIRST_DIFF_KEY, SECOND_DIFF_KEY } from "../const";
import { useGetFlowQuery } from "../api";

function Flow(flow1: string, flow2: string) {
  return (
    <div>
      <ReactDiffViewer
        oldValue={flow1}
        newValue={flow2}
        splitView={true}
        showDiffOnly={false}
        useDarkTheme={false}
        hideLineNumbers={true}
        styles={{
          line: {
            wordBreak: "break-word",
          },
        }}
      />
      <hr
        style={{
          height: "1px",
          color: "inherit",
          borderTopWidth: "5px",
        }}
      />
    </div>
  );
}

function isASCII(str: string) {
  return /^[\x00-\x7F]*$/.test(str);
}

const displayOptions = ["Plain", "Hex"];

// Derives the display mode for two given flows
const deriveDisplayMode = (
  firstFlow: FullFlow,
  secondFlow: FullFlow
): typeof displayOptions[number] => {
  if (firstFlow && secondFlow) {
    for (
      let i = 0;
      i < Math.min(firstFlow.flow.length, secondFlow.flow.length);
      i++
    ) {
      if (
        !isASCII(firstFlow.flow[0].flow[i].data) ||
        !isASCII(secondFlow.flow[0].flow[i].data)
      ) {
        return displayOptions[1];
      }
    }
  }

  return displayOptions[0];
};

export function DiffView() {
  let [searchParams] = useSearchParams();
  const firstFlowParam = searchParams.get(FIRST_DIFF_KEY);
  const firstFlowId = firstFlowParam?.split(":")[0];
  const firstFlowRepr = parseInt(firstFlowParam?.split(":")[1] ?? "0");
  const secondFlowParam = searchParams.get(SECOND_DIFF_KEY);
  const secondFlowId = secondFlowParam?.split(":")[0];
  const secondFlowRepr = parseInt(secondFlowParam?.split(":")[1] ?? "0");

  let { data: firstFlow, isLoading: firstFlowLoading, isError: firstFlowError } = useGetFlowQuery(
    firstFlowId!,
    {
      skip: firstFlowId === null,
    }
  );
  let { data: secondFlow, isLoading: secondFlowLoading, isError: secondFlowError } = useGetFlowQuery(
    secondFlowId!,
    {
      skip: secondFlowId === null,
    }
  );

  const [displayOption, setDisplayOption] = useState(
    deriveDisplayMode(firstFlow!, secondFlow!)
  );

  if (firstFlowError || secondFlowError) {
    return <div>Invalid flow id</div>;
  }

  if (firstFlowLoading || secondFlowLoading) {
    return <div>Loading...</div>;
  }

  return (
    <div>
      <div className="sticky shadow-md bg-white overflow-auto py-1 border-y flex items-center">
        <RadioGroup
          options={displayOptions}
          value={displayOption}
          onChange={setDisplayOption}
          className="flex gap-2 text-gray-800 text-sm mr-4"
        />
      </div>

      {/* Plain */}
      {displayOption === displayOptions[0] && (
        <div>
          {Array.from(
            {
              length: Math.min(firstFlow!.flow[firstFlowRepr].flow.length, secondFlow!.flow[secondFlowRepr].flow.length),
            },
            (_, i) => Flow(firstFlow!.flow[firstFlowRepr].flow[i].data, secondFlow!.flow[secondFlowRepr].flow[i].data)
          )}
        </div>
      )}

      {/* Hex */}
      {displayOption === displayOptions[1] && (
        <div>
          {Array.from(
            {
              length: Math.min(firstFlow!.flow[firstFlowRepr].flow.length, secondFlow!.flow[secondFlowRepr].flow.length),
            },
            (_, i) =>
              Flow(
                hexy(Buffer.from(firstFlow!.flow[firstFlowRepr].flow[i].b64, 'base64'), { format: "twos" }),
                hexy(Buffer.from(secondFlow!.flow[secondFlowRepr].flow[i].b64, 'base64'), { format: "twos" })
              )
          )}
        </div>
      )}
    </div>
  );
}
