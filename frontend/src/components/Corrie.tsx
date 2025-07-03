import { useSearchParams, useNavigate } from "react-router-dom";
import { useCallback } from "react";
import { Flow, Stats, TicksAttackInfo } from "../types";
import {
  SERVICE_FILTER_KEY,
  TEXT_FILTER_KEY,
  START_FILTER_KEY,
  END_FILTER_KEY,
  CORRELATION_MODE_KEY,
  FLOW_LIST_REFETCH_INTERVAL_MS,
  UNDER_ATTACK_REFETCH_INTERVAL_MS,
} from "../const";
import useDebounce from "../hooks/useDebounce";

import ReactApexChart from "react-apexcharts";
import { ApexOptions } from "apexcharts";
import {
  useGetFlowsQuery,
  useGetServicesQuery,
  useGetStatsQuery,
  useGetUnderAttackQuery
} from "../api";
import { getTickStuff } from "../tick";
import { useAppSelector } from "../store";
import { tagToColor } from "./Tag";

interface TickInfoData {
  startTick: number;
  endTick: number;
  flagLifetime: number;
  tickToUnixTime: (a: number) => number;
  unixTimeToTick: (a: number) => number;
}
interface GraphProps {
  flowList: Flow[];
  statsList: Stats[];
  underAttackData: TicksAttackInfo;
  mode: string;
  searchParams: URLSearchParams;
  setSearchParams: (a: URLSearchParams) => void;
  onClickNavigate: (a: string) => void;
  tickInfoData: TickInfoData
}

export const Corrie = () => {
  const { data: services } = useGetServicesQuery();
  const includeTags = useAppSelector((state) => state.filter.includeTags);
  const excludeTags = useAppSelector((state) => state.filter.excludeTags);
  const filterFlags = useAppSelector((state) => state.filter.filterFlags);
  const filterFlagids = useAppSelector((state) => state.filter.filterFlagids);
  const tagIntersectionMode = useAppSelector((state) => state.filter.tagIntersectionMode);

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

  let { currentTick, flagLifetime, startTickParam, endTickParam, unixTimeToTick, tickToUnixTime } = getTickStuff();

  let startTick = startTickParam ?? 0;
  let endTick = endTickParam ?? currentTick;
  if (startTick < 0) {
    startTick = 0;
  }
  if (endTick < startTick) {
    endTick = startTick;
  }

  const needsStats = mode == "flags" || mode == "tags";

  const statsData = needsStats ? useGetStatsQuery(
    {
      service: service_name,
      tick_from: startTick,
      tick_to: endTick,
    }
  ).data : [];

  const flowData = !needsStats ? useGetFlowsQuery(
    {
      regex_insensitive: debounced_text_filter,
      ip_dst: service?.ip,
      port_dst: service?.port,
      time_from: from_filter ? new Date(parseInt(from_filter)).toISOString() : undefined,
      time_to: to_filter ? new Date(parseInt(to_filter)).toISOString() : undefined,
      tags_include: includeTags,
      tags_exclude: excludeTags,
      tag_intersection_mode: tagIntersectionMode,
      flags: filterFlags,
      flagids: filterFlagids,
    },
    {
      refetchOnMountOrArgChange: true,
      pollingInterval: FLOW_LIST_REFETCH_INTERVAL_MS,
    }
  ).data : [];

  // TODO: this fetches under attack data always - not sure how to fetch it only in under-attack mode due to react hooks having to be called in same order always
  const underAttackData = useGetUnderAttackQuery(
    {
      from_tick: startTick,
      to_tick: endTick + flagLifetime,
    },
    {
      pollingInterval: UNDER_ATTACK_REFETCH_INTERVAL_MS,
    }
  ).data;

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
    underAttackData: underAttackData || {},
    mode: mode,
    searchParams: searchParams,
    setSearchParams: setSearchParams,
    onClickNavigate: onClickNavigate,
    tickInfoData: { startTick, endTick, flagLifetime, unixTimeToTick, tickToUnixTime },
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
          <button
            className={mode == "under-attack" ? activeButtonClass : inactiveButtonClass}
            onClick={() => setCorrelationMode("under-attack")}
          >
            under attack
          </button>
        </div>
      </div>
      <div className="flex-1 w-full overflow-hidden p-4">
        {(mode == "packets" || mode == "time") && TimePacketGraph(graphProps)}
        {mode == "volume" && VolumeGraph(graphProps)}
        {(mode == "tags" || mode == "flags") && BarPerTickGraph(graphProps, mode)}
        {(mode == "under-attack") && UnderAttackGraph(graphProps)}
      </div>
    </div>
  );
};

function BarPerTickGraph(graphProps: GraphProps, mode: string) {
  const statsList = graphProps.statsList;
  const searchParams = graphProps.searchParams;
  const setSearchParams = graphProps.setSearchParams;
  let startTick = graphProps.tickInfoData.startTick;
  let endTick = graphProps.tickInfoData.endTick;
  let tickToUnixTime = graphProps.tickInfoData.tickToUnixTime;

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

  const colors: any = {
    "tag_flag_in": tagToColor("flag-in"),
    "tag_flag_out": tagToColor("flag-out"),
    "tag_enemy": tagToColor("enemy"),
    "tag_blocked": tagToColor("blocked"),
    "tag_suricata": tagToColor("suricata"),

    "flag_in": tagToColor("flag-in"),
    "flag_out": tagToColor("flag-out"),
  };

  Object.keys(colors).forEach(t => {
    if ((mode == "tags" && t.startsWith("tag_")) || (mode == "flags" && t.startsWith("flag_"))) {
      const data = Array(endTick - startTick).fill(0);

      statsList.forEach(s => {
        data[s.tick - startTick] = s[t];
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
      return flow.id;
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
      return flow.id;
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

function UnderAttackGraph(graphProps: GraphProps) {
  const underAttackData = graphProps.underAttackData;
  const tickInfoData = graphProps.tickInfoData;
  const tickToUnixTime = tickInfoData.tickToUnixTime;
  const searchParams = graphProps.searchParams;
  const setSearchParams = graphProps.setSearchParams;

  const options: ApexOptions = {
    plotOptions: {
      bar: {
        horizontal: true,
        barHeight: '30%',
        rangeBarGroupRows: true,
      },
    },
    tooltip: {
      custom: (opts) => {
        if (opts.y1 === opts.y2 - 1) return `Tick ${opts.y1}`;

        return `Ticks ${opts.y1} - ${opts.y2 - 1}`;
      },
    },
    legend: {
      show: false,
    },
    xaxis: {
      min: tickInfoData.startTick,
      max: tickInfoData.endTick,
      tickAmount: Math.min(Math.abs(tickInfoData.startTick - tickInfoData.endTick), 25),
      decimalsInFloat: 0,
      title: {
        text: "Tick",
      },
    },
    yaxis: {
      tickAmount: 0,
    },
    chart: {
      animations: {
        enabled: false,
      },
      events: {
        dataPointSelection: (event, ctx, config) => {
          const y = config.w.config.series[config.seriesIndex].data[0].y;
          const start = Math.floor(tickToUnixTime(y[0]));
          const end = Math.ceil(tickToUnixTime(y[1]));
          searchParams.set(START_FILTER_KEY, start.toString());
          searchParams.set(END_FILTER_KEY, end.toString());
          setSearchParams(searchParams);
        },
        beforeZoom: function (chartContext, { xaxis }) {
          const start = Math.floor(tickToUnixTime(xaxis.min));
          const end = Math.ceil(tickToUnixTime(xaxis.max));
          searchParams.set(START_FILTER_KEY, start.toString());
          searchParams.set(END_FILTER_KEY, end.toString());
          setSearchParams(searchParams);
        },
      },
    }
  };

  // TODO: service names between visualizer and tulip don't necessarily match, how should we consider filters?
  const ranges: Record<string, { from_tick: number, to_tick: number }[]> = {};
  const lastSeen: Record<string, number | undefined> = {};
  for (const tick in underAttackData) {
    const tickNumber = Number(tick);
    let from_tick = Math.max(0, tickNumber - (tickInfoData.flagLifetime - 1));

    const services = underAttackData[tick];
    for (const service in services) {
      const value = services[service];
      if (value <= 0) continue;

      ranges[service] = ranges[service] || [];

      // Heuristic: if we had previous ticks where we lost the flags, most likely an attack occured afterward
      if (lastSeen[service] !== undefined) from_tick = Math.max(from_tick, lastSeen[service]!);

      ranges[service].push({
        from_tick: from_tick,
        to_tick: tickNumber + 1,
      })
      lastSeen[service] = tickNumber + 1;
    }
  }

  const series: ApexAxisChartSeries = [];
  for (const service in ranges) {
    for (const range of ranges[service]) {
      series.push({
        data: [
          {
            x: service,
            y: [range.from_tick, range.to_tick],
            goals: [
              {
                name: 'tick start',
                value: range.to_tick - 1,
                strokeColor: '#CD2F2A',
              },
              {
                name: 'tick end',
                value: range.to_tick,
                strokeColor: '#CD2F2A',
              }
            ],
          },
        ],
      })
    }
  }

  return <ReactApexChart options={options} series={series} type="rangeBar" />;
}
