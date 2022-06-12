import { format, parse } from "date-fns";
import { useAtom } from "jotai";
import { atomWithStorage } from "jotai/utils";
import { Suspense } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Service, useTulip } from "../api";

import {
  END_FILTER_KEY,
  SERVICE_FILTER_KEY,
  START_FILTER_KEY,
  TEXT_FILTER_KEY,
} from "../App";
import { useCTF } from "../pages/Home";

export const showHexAtom = atomWithStorage("showHex", false);

function ServiceSelection() {
  const FILTER_KEY = SERVICE_FILTER_KEY;

  // TODO add all, maybe user react-select

  const { api, services } = useTulip();

  const service_select = [
    {
      ip: "",
      port: 0,
      name: "all",
    },
    ...services,
  ];
  let [searchParams, setSearchParams] = useSearchParams();
  console.log(...searchParams.entries(), service_select);
  return (
    <select
      value={searchParams.get(FILTER_KEY) ?? ""}
      onChange={(event) => {
        let serviceFilter = event.target.value;
        if (serviceFilter && serviceFilter != "all") {
          searchParams.set(FILTER_KEY, serviceFilter);
        } else {
          searchParams.delete(FILTER_KEY);
        }
        setSearchParams(searchParams);
      }}
    >
      {service_select.map((service) => (
        <option key={service.name} value={service.name}>
          {service.name}
        </option>
      ))}
    </select>
  );
}

function TextSearch() {
  const FILTER_KEY = TEXT_FILTER_KEY;
  let [searchParams, setSearchParams] = useSearchParams();
  return (
    <div>
      <input
        type="text"
        placeholder="regex"
        value={searchParams.get(FILTER_KEY) || ""}
        onChange={(event) => {
          let textFilter = event.target.value;
          if (textFilter) {
            searchParams.set(FILTER_KEY, textFilter);
          } else {
            searchParams.delete(FILTER_KEY);
          }
          setSearchParams(searchParams);
        }}
      ></input>
    </div>
  );
}

function useMessyTimeStuff() {
  let [searchParams, setSearchParams] = useSearchParams();

  const { startDate, tickLength } = useCTF();

  function setTimeParam(startTick: string, param: string) {
    const parsedTick = startTick === "" ? undefined : parseInt(startTick);
    const unixTime = tickToUnixTime(parsedTick);
    if (unixTime) {
      searchParams.set(param, unixTime.toString());
    } else {
      searchParams.delete(param);
    }
    setSearchParams(searchParams);
  }

  const startTimeParamUnix = searchParams.get(START_FILTER_KEY);
  const endTimeParamUnix = searchParams.get(END_FILTER_KEY);

  function unixTimeToTick(unixTime: string | null): number | undefined {
    if (unixTime === null) {
      return;
    }
    let unixTimeInt = parseInt(unixTime);
    if (isNaN(unixTimeInt)) {
      return;
    }
    const tick = Math.floor(
      (unixTimeInt - new Date(startDate).valueOf()) / tickLength
    );

    return tick;
  }

  function tickToUnixTime(tick?: number) {
    if (!tick) {
      return;
    }
    const unixTime = new Date(startDate).valueOf() + tickLength * tick;
    return unixTime;
  }

  const startTick = unixTimeToTick(startTimeParamUnix);
  const endTick = unixTimeToTick(endTimeParamUnix);

  return {
    unixTimeToTick,
    startDate,
    tickLength,
    setTimeParam,
    startTick,
    endTick,
  };
}

function StartDateSelection() {
  const { setTimeParam, startTick } = useMessyTimeStuff();

  return (
    <div>
      <input
        className="w-20"
        type="number"
        placeholder="from"
        value={startTick}
        onChange={(event) => {
          setTimeParam(event.target.value, START_FILTER_KEY);
        }}
      ></input>
    </div>
  );
}

function EndDateSelection() {
  const { setTimeParam, endTick } = useMessyTimeStuff();

  return (
    <div>
      <input
        className="w-20"
        type="number"
        placeholder="to"
        value={endTick}
        onChange={(event) => {
          setTimeParam(event.target.value, END_FILTER_KEY);
        }}
      ></input>
    </div>
  );
}

function ShowHexToggle() {
  const [showHex, setShowHex] = useAtom(showHexAtom);

  return (
    <div className="flex items-baseline mx-4">
      <input
        type="checkbox"
        className="mr-2"
        checked={showHex}
        onChange={() => {
          setShowHex(!showHex);
        }}
      />
      <label htmlFor="">Hexdump</label>
    </div>
  );
}

export function Header() {
  let [searchParams] = useSearchParams();

  return (
    <>
      <Link to={`/?${searchParams}`}>
        <div className="header-icon">🌷</div>
      </Link>
      <div>
        <TextSearch></TextSearch>
      </div>
      <div>
        <Suspense>
          <ServiceSelection></ServiceSelection>
        </Suspense>
      </div>
      <div>
        <StartDateSelection></StartDateSelection>
      </div>
      <div>
        <EndDateSelection></EndDateSelection>
      </div>

      {/* <div className="ml-auto">
        <ShowHexToggle></ShowHexToggle>
      </div> */}
    </>
  );
}
