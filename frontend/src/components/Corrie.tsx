import { useSearchParams, useParams, useNavigate } from "react-router-dom";
import { useCallback } from "react";
import { Flow, Stats } from "../types";
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
import { useGetFlowsQuery, useGetServicesQuery, useGetTickInfoQuery, useGetStatsQuery } from "../api";
import { TICK_REFETCH_INTERVAL_MS } from "../const";
import { TickInfo } from "../types";
import { useAppSelector } from "../store";
import { tagToColor } from "./Tag";

interface TickInfoWithTimeStuff extends TickInfo {
  startTick: number;
  endTick: number;
  tickToUnixTime: (a: number) => number;
  unixTimeToTick: (a: number) => number;
}
interface GraphProps {
  flowList: Flow[];
  statsList: Stats[];
  mode: string;
  searchParams: URLSearchParams;
  setSearchParams: (a: URLSearchParams) => void;
  onClickNavigate: (a: string) => void;
  tickInfoData: TickInfoWithTimeStuff;
}

// TODO find a better way to do this
function getTickStuffFromTimeParams(tickInfoData: TickInfo | undefined, searchParams: URLSearchParams) {
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
  
  if (startTick < 0) {
    startTick = 0;
  }
  
  if (endTick < startTick) {
    endTick = startTick;
  }
  
  return { startTick, endTick, unixTimeToTick, tickToUnixTime };
}

export const Corrie = () => {
  const { data: services } = useGetServicesQuery();
  const includeTags = useAppSelector((state) => state.filter.includeTags);
  const excludeTags = useAppSelector((state) => state.filter.excludeTags);

  const [searchParams, setSearchParams] = useSearchParams();

  const service_name = searchParams.get(SERVICE_FILTER_KEY) ?? "";
  const service = services && services.find((s) => s.name == service_name);

  const text_filter = searchParams.get(TEXT_FILTER_KEY) ?? undefined;
  const from_filter = searchParams.get(START_FILTER_KEY) ?? undefined;
  const to_filter = searchParams.get(END_FILTER_KEY) ?? undefined;

  const debounced_text_filter = useDebounce(text_filter, 300);

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
  
  const tickStuff = getTickStuffFromTimeParams(tickInfoData, searchParams)
  
  const needsStats = mode == "flags" || mode == "tags";

  const statsData = needsStats ? useGetStatsQuery(
    {
      service: service_name,
      from_tick: tickStuff.startTick,
      to_tick: tickStuff.endTick,
    }
  ).data : [];

  const flowData = !needsStats ? useGetFlowsQuery(
    {
      "flow.data": debounced_text_filter,
      dst_ip: service?.ip,
      dst_port: service?.port,
      from_time: from_filter,
      to_time: to_filter,
      service: "", // FIXME
      includeTags: includeTags,
      excludeTags: excludeTags,
    },
    {
      refetchOnMountOrArgChange: true,
      pollingInterval: FLOW_LIST_REFETCH_INTERVAL_MS,
    }
  ).data : [];

  // TODO: fix the below transformation - move it to server
  // Diederik gives you a beer once it has been fixed
  const transformedFlowData = flowData?.map((flow) => ({
    ...flow,
    service_tag:
      services?.find((s) => s.ip === flow.dst_ip && s.port === flow.dst_port)
        ?.name ?? "unknown",
  }));

  const graphProps: GraphProps = {
    flowList: transformedFlowData || [],
    statsList: statsData || [],
    mode: mode,
    searchParams: searchParams,
    setSearchParams: setSearchParams,
    onClickNavigate: onClickNavigate,
    tickInfoData: Object.assign(tickStuff, tickInfoData)
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
            className={mode == "tags" ? activeButtonClass : inactiveButtonClass}
            onClick={() => setCorrelationMode("tags")}
          >
            tags
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
        {(mode == "tags" || mode == "flags") && BarPerTickGraph(graphProps, mode)}
      </div>
    </div>
  );
};

function BarPerTickGraph(graphProps: GraphProps, mode: string) {
  const statsList = graphProps.statsList;
  const searchParams = graphProps.searchParams;
  const setSearchParams = graphProps.setSearchParams;
  const tickInfoData = graphProps.tickInfoData;
  let startTick = tickInfoData.startTick;
  let endTick = tickInfoData.endTick;
  const tickToUnixTime = tickInfoData.tickToUnixTime;

  const SEARCH_CAP = 50;
  const DEFAULT_CAP = 15;

  // Hard limit for performance reasons
  if (searchParams.has(START_FILTER_KEY) && searchParams.has(END_FILTER_KEY)) {
    startTick = Math.max(Math.max(0, startTick), endTick - SEARCH_CAP);
  } else if (endTick - startTick > DEFAULT_CAP) {
    startTick = Math.max(0, endTick - DEFAULT_CAP);
  }

  var options: ApexOptions = {
    plotOptions: {
      bar: {
        horizontal: false,
        columnWidth: "90%",
      }
    },
    grid: {
      position: "back",
      xaxis: {
        lines: {
          show: endTick !== startTick + 1
        }
      },
      yaxis: {
        lines: {
          show: false
        }
      }
    },
    dataLabels: {
      enabled: false,
    },
    stroke: {
      show: true,
      width: 2,
      colors: ['transparent']
    },
    xaxis: {
      categories: Array.from({ length: endTick - startTick }, (_, i) => startTick + i),
      title: {
        text: "Ticks"
      }
    },
    yaxis: {
      title: {
        text: "Number of flows"
      }
    },
    tooltip: {
      x: {
        formatter: function (v) {
          return "Tick " + v;
        }
      }
    },
    chart: {
      animations: {
        enabled: false
      },
      events: {
        click: function (e, chartContext, options) {
          const tick = options.dataPointIndex;
          if (tick !== -1) {
            const start = Math.floor(tickToUnixTime(tick + startTick));
            const end = Math.ceil(tickToUnixTime(tick + startTick + 1));
            searchParams.set(START_FILTER_KEY, start.toString());
            searchParams.set(END_FILTER_KEY, end.toString());
            setSearchParams(searchParams);
          }
        },
      },
    },
  };
  
  let series: ApexAxisChartSeries = [];
  
  const colors : any = {
    "tag_flag_in": tagToColor("flag-in"),
    "tag_flag_out": tagToColor("flag-out"),
    "tag_enemy": tagToColor("enemy"),
    "tag_blocked": tagToColor("blocked"),
    "tag_suricata": tagToColor("suricata"),

    "flag_in": tagToColor("flag-in"),
    "flag_out":tagToColor("flag-out"),
  };
  
  Object.keys(colors).forEach(t => {
    if ((mode == "tags" && t.startsWith("tag_")) || (mode == "flags" && t.startsWith("flag_"))) {
      const data = Array(endTick - startTick).fill(0);

      statsList.forEach(s => {
        data[s._id - startTick] = s[t];
      });

      series.push({
        name: t,
        data: data,
        color: colors[t]
      });
    }
  });
 
  return (
      <ReactApexChart
        options={options}
        series={series}
        type="bar"
        height="100%"
        width="100%"
      />
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
