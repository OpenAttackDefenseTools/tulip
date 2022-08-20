import {
    useSearchParams,
    useParams,
} from "react-router-dom";
import { useEffect, useState } from "react";
import { useAtom } from "jotai";
import { Flow, useTulip } from "../api";
import {
    SERVICE_FILTER_KEY,
    TEXT_FILTER_KEY,
    START_FILTER_KEY,
    END_FILTER_KEY,
} from "../App";
import useDebounce from "../hooks/useDebounce";
import { lastRefreshAtom } from "./Header";

import Chart from "react-apexcharts";

export const Corrie = () => {
    let [searchParams] = useSearchParams();
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

    console.log(flowList);

    let state = {

        series: [{
            name: 'Flows',
            data: flowList.map((flow) => {
                return [flow.time, flow.duration]
            })
        },
        ],
        options: {
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
            }
        },
    };
    return (
        <div>
            {/* <div>Corrie data:</div> */}
            <Chart
                options={state.options}
                series={state.series}
                type="scatter"
            />
        </div>
    );
}
