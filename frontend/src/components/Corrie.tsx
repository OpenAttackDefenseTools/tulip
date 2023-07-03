import { useSearchParams, useParams, useNavigate } from "react-router-dom";
import { useCallback } from "react";
import { Flow } from "../types";
import {
  SERVICE_FILTER_KEY,
  TEXT_FILTER_KEY,
  START_FILTER_KEY,
  END_FILTER_KEY,
  CORRELATION_MODE_KEY,
  FLOW_LIST_REFETCH_INTERVAL_MS,
} from "../const";
import useDebounce from "../hooks/useDebounce";

import ReactApexChart from "react-apexcharts";
import { ApexOptions } from "apexcharts";
import { useGetFlowsQuery, useGetServicesQuery, useGetTickInfoQuery } from "../api";
import { TICK_REFETCH_INTERVAL_MS } from "../const";
import { TickInfo } from "../types";
import { useAppSelector } from "../store";

interface GraphProps {
  flowList: Flow[];
  mode: string;
  searchParams: URLSearchParams;
  setSearchParams: (a: URLSearchParams) => void;
  onClickNavigate: (a: string) => void;
  tickInfoData: TickInfo | undefined;
}

export const Corrie = () => {
  const { data: services } = useGetServicesQuery();
  const filterTags = useAppSelector((state) => state.filter.filterTags);

  const [searchParams, setSearchParams] = useSearchParams();

  const service_name = searchParams.get(SERVICE_FILTER_KEY) ?? "";
  const service = services && services.find((s) => s.name == service_name);

  const text_filter = searchParams.get(TEXT_FILTER_KEY) ?? undefined;
  const from_filter = searchParams.get(START_FILTER_KEY) ?? undefined;
  const to_filter = searchParams.get(END_FILTER_KEY) ?? undefined;

  const debounced_text_filter = useDebounce(text_filter, 300);

  const { data: flowData, isLoading } = useGetFlowsQuery(
    {
      "flow.data": debounced_text_filter,
      dst_ip: service?.ip,
      dst_port: service?.port,
      from_time: from_filter,
      to_time: to_filter,
      service: "", // FIXME
      tags: filterTags,
    },
    {
      refetchOnMountOrArgChange: true,
      pollingInterval: FLOW_LIST_REFETCH_INTERVAL_MS,
    }
  );

  // TODO: fix the below transformation - move it to server
  // Diederik gives you a beer once it has been fixed
  const transformedFlowData = flowData?.map((flow) => ({
    ...flow,
    service_tag:
      services?.find((s) => s.ip === flow.dst_ip && s.port === flow.dst_port)
        ?.name ?? "unknown",
  }));

  const mode = searchParams.get("correlation") ?? "time";
  const setCorrelationMode = (mode: string) => {
    searchParams.set(CORRELATION_MODE_KEY, mode);
    setSearchParams(searchParams);
  };

  const inactiveButtonClass = "bg-blue-100 text-gray-800 rounded-md px-2 py-1";
  const activeButtonClass = `${inactiveButtonClass} ring-2 ring-gray-500`;

  const navigate = useNavigate();
  const onClickNavigate = useCallback(
    (loc: string) => navigate(loc, { replace: true }),
    [navigate]
  );

  // TODO find a better way to do this
  const { data: tickInfoData } = useGetTickInfoQuery(undefined, {
    pollingInterval: TICK_REFETCH_INTERVAL_MS,
  });

  const graphProps: GraphProps = {
    flowList: transformedFlowData || [],
    mode: mode,
    searchParams: searchParams,
    setSearchParams: setSearchParams,
    onClickNavigate: onClickNavigate,
    tickInfoData: tickInfoData
  };

  return (
    <div className="flex flex-col h-full">
      <div className="text-sm bg-white border-b-gray-300 border-b shadow-md flex flex-col">
        <div className="p-2 flex space-x-2" style={{ height: 50 }}>
          <a className="text-center px-2 py-2">Correlation mode: </a>
          <button
            className={mode == "time" ? activeButtonClass : inactiveButtonClass}
            onClick={() => setCorrelationMode("time")}
          >
            time
          </button>
          <button
            className={
              mode == "packets" ? activeButtonClass : inactiveButtonClass
            }
            onClick={() => setCorrelationMode("packets")}
          >
            packets
          </button>
          <button
            className={mode == "volume" ? activeButtonClass : inactiveButtonClass}
            onClick={() => setCorrelationMode("volume")}
          >
            volume
          </button>
          <button
            className={mode == "flags" ? activeButtonClass : inactiveButtonClass}
            onClick={() => setCorrelationMode("flags")}
          >
            flags
          </button>
        </div>
      </div>
      <div className="flex-1 w-full overflow-hidden p-4">
        {(mode == "packets" || mode == "time") && TimePacketGraph(graphProps)}
        {mode == "volume" && VolumeGraph(graphProps)}
        {mode == "flags" && FlagsGraph(graphProps)}
      </div>
    </div>
  );
};

function FlagsGraph(graphProps: GraphProps) {
  const flowList = graphProps.flowList;
  const searchParams = graphProps.searchParams;
  const setSearchParams = graphProps.setSearchParams;
  const tickInfoData = graphProps.tickInfoData;

  const SEARCH_CAP = 100;
  const DEFAULT_CAP = 25;

  // TODO find a better way to do this
  const startDate = tickInfoData?.startDate ?? "1970-01-01T00:00:00Z";
  const tickLength = tickInfoData?.tickLength ?? 1000;

  function unixTimeToTick(unixTimeInt: number): number {
    return Math.floor(
      (unixTimeInt - new Date(startDate).valueOf()) / tickLength
    );
  }

  function tickToUnixTime(tick: number): number {
    return new Date(startDate).valueOf() + tickLength * tick;
  }

  let endTick = Math.ceil(unixTimeToTick(parseInt(searchParams.get(END_FILTER_KEY) ?? new Date().valueOf().toString())));
  let startTick = Math.floor(unixTimeToTick(parseInt(searchParams.get(START_FILTER_KEY) ?? new Date(startDate).valueOf().toString())));

  // Hard limit for performance reasons
  if (searchParams.has(START_FILTER_KEY) && searchParams.has(END_FILTER_KEY)) {
    startTick = Math.max(Math.max(0, startTick), endTick - SEARCH_CAP);
  } else if (endTick - startTick > DEFAULT_CAP) {
    startTick = Math.max(0, endTick - DEFAULT_CAP);
  }

  let flags: any = {
    in: {},
    out: {}
  };

  for (var i = startTick; i <= endTick; i++) {
      flags.in[i] = {
        x: i, y: 0
      }

      flags.out[i] = {
        x: i, y: 0
      }
  }

  flowList.forEach((flow) => {
    const tick = unixTimeToTick(flow.time);
    
    if (tick < startTick || tick > endTick) {
      return;
    }

    if (flow.tags.includes("flag-in")) {
      flags.in[tick].y++;
    }

    if (flow.tags.includes("flag-out")) {
      flags.in[tick].y++;
    }
  });

  var options: ApexOptions = {
    dataLabels: {
      enabled: false,
    },
    grid: {
      xaxis: {
        lines: {
          show: true,
        },
      },
      yaxis: {
        lines: {
          show: true,
        },
      },
    },
    xaxis: {
      type: "numeric"
    },
    chart: {
      animations: {
        enabled: false,
      },
      events: {
        beforeZoom: function (chartContext, { xaxis }) {
          const start = Math.floor(tickToUnixTime(xaxis.min));
          const end = Math.ceil(tickToUnixTime(xaxis.max + 1));
          searchParams.set(START_FILTER_KEY, start.toString());
          searchParams.set(END_FILTER_KEY, end.toString());
          setSearchParams(searchParams);
        },
      },
    },
  };

  const series1: ApexAxisChartSeries = [
    {
      name: "Flag In",
      data: Object.values(flags.in)
    }
  ];

  const series2: ApexAxisChartSeries = [
    {
      name: "Flag Out",
      data: Object.values(flags.out)
    }
  ];
 
  // TODO remove hardcoded height values and find a way to split this
  return (
    <div id="chart-wrapper">
      <div id="chart-flag-in">
        <ReactApexChart
          options={Object.assign({ labels: Object.keys(flags.in), title: { text: "Flag In by Flow", align: "left" } }, options)}
          series={series1}
          type="line"
          height={360}
          width="100%"
        />
      </div>
      <div id="chart-flag-out">
        <ReactApexChart
          options={Object.assign({ labels: Object.keys(flags.out), title: { text: "Flag Out by Flow", align: "left" } }, options)}
          series={series2}
          type="line"
          height={360}
          width="100%"
        />
      </div>
    </div>
  );
}

function TimePacketGraph(graphProps: GraphProps) {
  const flowList = graphProps.flowList;
  const mode = graphProps.mode;
  const searchParams = graphProps.searchParams;
  const setSearchParams = graphProps.setSearchParams;
  const onClickNavigate = graphProps.onClickNavigate;

  const series: ApexAxisChartSeries = [
    {
      name: "Flows",
      data: flowList.map((flow) => {
        let y = flow.duration;
        if (mode == "packets") {
          y = flow.num_packets;
        }
        return { x: flow.time, y: y };
      }),
    },
  ];

  const options: ApexOptions = {
    dataLabels: {
      enabled: false,
    },
    grid: {
      xaxis: {
        lines: {
          show: true,
        },
      },
      yaxis: {
        lines: {
          show: true,
        },
      },
    },
    xaxis: {
      type: "datetime", // FIXME: Timezone is not displayed correctly
    },
    labels: flowList.map((flow) => {
      return flow._id.$oid;
    }),
    chart: {
      animations: {
        enabled: false,
      },
      events: {
        dataPointSelection: (event: any, chartContext: any, config: any) => {
          // Retrieve flowList from chart's labels. This is hacky, refer to FIXME above.
          const flowIdList = config.w.config.labels;
          const flow = flowIdList[config.dataPointIndex];
          onClickNavigate(`/flow/${flow}?${searchParams}`);
        },
        beforeZoom: function (chartContext, { xaxis }) {
          const start = Math.floor(xaxis.min);
          const end = Math.ceil(xaxis.max);
          searchParams.set(START_FILTER_KEY, start.toString());
          searchParams.set(END_FILTER_KEY, end.toString());
          setSearchParams(searchParams);
        },
      },
    },
  };

  return (
    <ReactApexChart
      options={options}
      series={series}
      type="scatter"
      width="100%"
      height="100%"
    />
  );
}

function VolumeGraph(graphProps: GraphProps) {
  const flowList = graphProps.flowList;
  const mode = graphProps.mode;
  const searchParams = graphProps.searchParams;
  const setSearchParams = graphProps.setSearchParams;

  function chunkData(flowList: Flow[]) {
    let ret: any = [];
    let ts = 0;
    let acc = 0;
    const window_size = 30000;
    flowList.forEach((flow) => {
      if (ts == 0) {
        ts = flow.time;
      }

      if (ts - flow.time > window_size) {
        ret.push({ x: ts, y: acc });
        ts = 0;
        acc = 0;
      } else {
        acc++;
      }
    });

    return ret;
  }

  const series_out: ApexAxisChartSeries = [
    {
      name: "Volume",
      data: chunkData(flowList),
    },
  ];

  const options: ApexOptions = {
    dataLabels: {
      enabled: false,
    },
    grid: {
      xaxis: {
        lines: {
          show: true,
        },
      },
      yaxis: {
        lines: {
          show: true,
        },
      },
    },
    xaxis: {
      type: "datetime", // FIXME: Timezone is not displayed correctly
    },
    labels: flowList.map((flow) => {
      return flow._id.$oid;
    }),
    chart: {
      animations: {
        enabled: false,
      },
      events: {
        beforeZoom: function (chartContext, { xaxis }) {
          const start = Math.floor(xaxis.min);
          const end = Math.ceil(xaxis.max);
          searchParams.set(START_FILTER_KEY, start.toString());
          searchParams.set(END_FILTER_KEY, end.toString());
          setSearchParams(searchParams);
        },
      },
    },
  };

  return <ReactApexChart options={options} series={series_out} type="line" />;
}
