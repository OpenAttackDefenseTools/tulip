import { useSearchParams } from "react-router-dom";

import {
  END_FILTER_KEY,
  START_FILTER_KEY,
  TICK_REFETCH_INTERVAL_MS,
} from "./const";
import { useGetTickInfoQuery } from "./api";

export function getTickStuff() {
  const { data: tickInfoData } = useGetTickInfoQuery(undefined, {
    pollingInterval: TICK_REFETCH_INTERVAL_MS,
  });
  const startDate = tickInfoData?.startDate ?? "1970-01-01T00:00:00Z";
  const tickLength = tickInfoData?.tickLength ?? 1000;
  const flagLifetime = tickInfoData?.flagLifetime ?? 0;
  const currentTick = unixTimeToTick(new Date().valueOf());

  function tickToUnixTime(tick: number): number {
    return new Date(startDate).valueOf() + tickLength * tick;;
  }

  function unixTimeToTick(unixTime: number): number {
    return Math.floor(
      (unixTime - new Date(startDate).valueOf()) / tickLength
    );
  }

  let [searchParams, setSearchParams] = useSearchParams();
  const startTimeParam = searchParams.get(START_FILTER_KEY);
  const endTimeParam = searchParams.get(END_FILTER_KEY);
  const startTimeParamUnix = startTimeParam === null ? undefined : parseInt(startTimeParam);
  const endTimeParamUnix = endTimeParam === null ? undefined : parseInt(endTimeParam);
  const startTickParam = startTimeParamUnix === undefined ? undefined : unixTimeToTick(startTimeParamUnix);
  const endTickParam = endTimeParamUnix === undefined ? undefined : unixTimeToTick(endTimeParamUnix);

  function setTimeParam(startTick: number | null, param: string) {
    if (startTick === null) {
      searchParams.delete(param);
    } else {
      searchParams.set(param, tickToUnixTime(startTick).toString());
    }
    setSearchParams(searchParams);
  }

  function setToLastnTicks(n: number) {
    const startTick = (currentTick ?? 0) - n;
    const endTick = (currentTick ?? 0) + 1; // to be sure
    setTimeParam(startTick, START_FILTER_KEY);
    setTimeParam(endTick, END_FILTER_KEY);
  }

  return {
    startDate,
    tickLength,
    flagLifetime,
    currentTick,
    tickToUnixTime,
    unixTimeToTick,
    startTickParam,
    endTickParam,
    setTimeParam,
    setToLastnTicks,
  }

}
