import { useSearchParams, Link, useParams } from "react-router-dom";
import React, { useEffect, useState } from "react";
import { useTulip, FlowData, FullFlow } from "../api";
import { Buffer } from "buffer";
import ReactDiffViewer from 'react-diff-viewer';
import {RadioGroup, RadioGroupProps} from "../components/RadioGroup"

import {
    ArrowCircleLeftIcon,
    ArrowCircleRightIcon,
} from "@heroicons/react/solid";
import { format } from "date-fns";
import classNames from "classnames";

import { hexy } from "hexy";
import { useCopy } from "../hooks/useCopy";

import {
    FIRST_DIFF_KEY,
    SECOND_DIFF_KEY,
} from "../App";


function Header() { }

function Flow(flow1: string, flow2: string) {
    return (
        <div>
        <ReactDiffViewer oldValue={flow1} newValue={flow2} splitView={true} showDiffOnly={false} useDarkTheme={false} hideLineNumbers={true}
            styles={{
                "line": {
                    "wordBreak": 'break-word',
                }
            }} />
            <hr style={{"height": "1px",
            "color": "inherit",
            "borderTopWidth": "5px"}}/>
        </div>
    );
}

function isASCII(str: string) {
    return /^[\x00-\x7F]*$/.test(str);
}

export function DiffView() {
    const displayOptions = ["Plain", "Hex"];

    let [searchParams] = useSearchParams();
    const params = useParams();
    const [flow1, setFlow1] = useState<FullFlow>();
    const [flow2, setFlow2] = useState<FullFlow>();
    const [displayOption, setDisplayOption] = useState("Plain");



    const flowId1 = searchParams.get(FIRST_DIFF_KEY) ?? "";
    const flowId2 = searchParams.get(SECOND_DIFF_KEY) ?? "";
    if (flowId1 == "" || flowId2 == "") {
        return <div>Invalid flow id</div>;
    }
    const id = params.id;

    const { api } = useTulip();

    useEffect(() => {
        const fetchData = async () => {

            let flowIds1 = await api.getFlow(flowId1);
            let flowIds2 = await api.getFlow(flowId2);

            for (let i = 0; i < Math.min(flowIds1.flow.length, flowIds2.flow.length); i++) {
                if (!isASCII(flowIds1.flow[i].data) || !isASCII(flowIds2.flow[i].data)) {
                    setDisplayOption("Hex");
                    break
                }
            }

            setFlow1(flowIds1);
            setFlow2(flowIds2);
        };
        fetchData().catch(console.error);
    }, [setDisplayOption, flow1, flow2]);


    if (flow1 === undefined || flow2 === undefined) {
        return <div>Loading...</div>;
    }



    return (
    <div><div
        className="sticky shadow-md bg-white overflow-auto py-1 border-y flex items-center">
        <RadioGroup
            options={displayOptions}
            value={displayOption}
            onChange={setDisplayOption}
            className="flex gap-2 text-gray-800 text-sm mr-4"
          />
        </div>
        <div>{Array.from({ length: Math.min(flow1.flow.length, flow2.flow.length) }, (_, i) => displayOption === "Hex" ? Flow(hexy(flow1.flow[i].data), hexy(flow2.flow[i].data)) :  Flow(flow1.flow[i].data, flow2.flow[i].data))}</div>
        </div>
    );
}
