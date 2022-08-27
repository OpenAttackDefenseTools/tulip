import {
    useSearchParams,
    useParams,
    useNavigate,
} from "react-router-dom";
import { useEffect, useState, useCallback } from "react";
import { useAtom } from "jotai";
import { Flow, useTulip } from "../api";
import {
    SERVICE_FILTER_KEY,
    TEXT_FILTER_KEY,
    START_FILTER_KEY,
    END_FILTER_KEY,
    CORRELATION_MODE_KEY,
} from "../App";
import useDebounce from "../hooks/useDebounce";
import { lastRefreshAtom } from "./Header";

import ReactApexChart from "react-apexcharts";
import { ApexOptions } from "apexcharts";

import ReactDOMServer from "react-dom/server";

import { FlowListEntry } from "./FlowList";

import classes from "./FlowList.module.css";
import classNames from "classnames";

export const Corrie = () => {
    let [searchParams, setSearchParams] = useSearchParams();
    let params = useParams();

    const { services, api, getFlows } = useTulip();

    const [flowList, setFlowList] = useState<Flow[]>([]);

    const service_name = searchParams.get(SERVICE_FILTER_KEY) ?? "";
    const service = services.find((s) => s.name == service_name);

    const text_filter = searchParams.get(TEXT_FILTER_KEY) ?? undefined;
    const from_filter = searchParams.get(START_FILTER_KEY) ?? undefined;
    const to_filter = searchParams.get(END_FILTER_KEY) ?? undefined;

    const debounced_text_filter = useDebounce(text_filter, 300);

    const [selectedTags, setSelectedTags] = useState<string[]>([]);
    const [availableTags, setAvailableTags] = useState<string[]>([]);

    const [loading, setLoading] = useState(false);

    const [lastRefresh, setLastRefresh] = useAtom(lastRefreshAtom);

    const navigate = useNavigate();
    const onClickNavicate = useCallback((loc: string) => navigate(loc, { replace: true }), [navigate]);


    useEffect(() => {
        const fetchData = async () => {
            const data = await api.getTags();
            setAvailableTags(data);
            console.log(data);
        };
        fetchData().catch(console.error);
    }, []);

    useEffect(() => {
        const fetchData = async () => {
            setLoading(true);
            const data = await getFlows({
                "flow.data": debounced_text_filter,
                dst_ip: service?.ip,
                dst_port: service?.port,
                from_time: from_filter,
                to_time: to_filter,
                service: "", // FIXME
                tags: selectedTags,
            });
            setFlowList(data);
            setLoading(false);
        };
        fetchData().catch(console.error);
    }, [
        service,
        debounced_text_filter,
        from_filter,
        to_filter,
        selectedTags,
        lastRefresh,
    ]);

    const mode = searchParams.get("correlation") ?? "time";

    const series: ApexAxisChartSeries = [{
        name: 'Flows',
        data: flowList.map((flow) => {
            let y = flow.duration;
            if (mode == "packets") {
                y = flow.num_packets
            }
            return { "x": flow.time, "y": y }
        })
    }];

    const options: ApexOptions = {
        dataLabels: {
            enabled: false
        },
        grid: {
            xaxis: {
                lines: {
                    show: true
                }
            },
            yaxis: {
                lines: {
                    show: true
                }
            },
        },
        xaxis: {
            type: 'datetime', // FIXME: Timezone is not displayed correctly
        },
        labels: flowList.map((flow) => { return flow._id.$oid; }),
        chart: {
            animations: {
                enabled: false
            },
            events: {
                dataPointSelection: (event: any, chartContext: any, config: any) => {
                    // Retrieve flowList from chart's labels. This is hacky, refer to FIXME above.
                    const flowIdList = config.w.config.labels;
                    const flow = flowIdList[config.dataPointIndex];
                    onClickNavicate(`/flow/${flow}?${searchParams}`);
                }
            }
        },
        /*
        tooltip: {
            followCursor: true,
            // TODO; these types are hacky
            custom: function ({ dataPointIndex, w}: {dataPointIndex: number, w: any}) {
                // Display corresponding flow like in the sidebar
                const flowList = w.config.labels;
                const flow = flowList[dataPointIndex];
                const element = (
                    <div className={classNames({
                        [classes.list_container]: true,
                    })}>
                        <FlowListEntry
                            key={flow._id.$oid}
                            flow={flow}
                            isActive={flow._id.$oid === params.id}
                            onHeartClick={() => { }}
                        />
                    </div>
                );
                return ReactDOMServer.renderToString(element);;
            },
        }
        */
    };

    const setCorrelationMode = (mode: string) => {
        searchParams.set(CORRELATION_MODE_KEY, mode);
        setSearchParams(searchParams);
    }

    const inactiveButtonClass = "bg-blue-100 text-gray-800 rounded-md px-2 py-1";
    const activeButtonClass = `${inactiveButtonClass} ring-2 ring-gray-500`;

    return (
        <div>
            <div className="flex flex-col h-full">
                <div className="text-sm bg-white border-b-gray-300 border-b shadow-md flex flex-col">
                    <div className="p-2 flex space-x-2" style={{ height: 50 }}>
                        <a className="text-center px-2 py-2">Correlation mode: </a>
                        <button className={mode == "time" ? activeButtonClass : inactiveButtonClass}
                                onClick={() => setCorrelationMode("time")}>
                            time
                        </button>
                        <button className={mode == "packets" ? activeButtonClass : inactiveButtonClass}
                                onClick={() => setCorrelationMode("packets")}>
                            packets
                        </button>
                    </div>
                </div>
            </div>
            <div>
            </div>
            <div>
                <ReactApexChart
                    options={options}
                    series={series}
                    type="scatter"
                />
            </div>
        </div>
    );
}
