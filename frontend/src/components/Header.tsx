import { useAtom } from "jotai";
import { atomWithStorage } from "jotai/utils";
import { Suspense } from "react";
import { useSearchParams } from "react-router-dom";
import { Service } from "../api";
import {
  END_FILTER_KEY,
  SERVICE_FILTER_KEY,
  START_FILTER_KEY,
  TEXT_FILTER_KEY,
  fetchUrlAtom,
} from "../App";

export const showHexAtom = atomWithStorage("showHex", false);

function ServiceSelection() {
  const FILTER_KEY = SERVICE_FILTER_KEY;

  // TODO add all, maybe user react-select
  const [services] = useAtom(fetchUrlAtom);
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

function StartDateSelection() {
  const FILTER_KEY = START_FILTER_KEY;
  let [searchParams, setSearchParams] = useSearchParams();
  return (
    <div>
      <input
        type="time"
        value={searchParams.get(FILTER_KEY) ?? undefined}
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

function EndDateSelection() {
  const FILTER_KEY = END_FILTER_KEY;
  let [searchParams, setSearchParams] = useSearchParams();
  return (
    <div>
      <input
        type="time"
        value={searchParams.get(FILTER_KEY) ?? undefined}
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
  return (
    <>
      <div className="header-icon">🌷</div>
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
