import { format, parse } from "date-fns";
import { Suspense, useState } from "react";
import { useHotkeys } from 'react-hotkeys-hook';
import {
  Link,
  useParams,
  useSearchParams,
  useNavigate,
} from "react-router-dom";
import ReactDiffViewer from "react-diff-viewer";

import {
    END_FILTER_KEY,
    SERVICE_FILTER_KEY,
    START_FILTER_KEY,
    TEXT_FILTER_KEY,
    FIRST_DIFF_KEY,
    SECOND_DIFF_KEY,
    SERVICE_REFETCH_INTERVAL_MS,
    TICK_REFETCH_INTERVAL_MS, SIMILARITY_FILTER_KEY,
} from "../const";
import {
  useGetFlowQuery,
  useGetServicesQuery,
  useGetTickInfoQuery,
} from "../api";

function ServiceSelection() {
  const FILTER_KEY = SERVICE_FILTER_KEY;

  // TODO add all, maybe user react-select

  const { data: services } = useGetServicesQuery(undefined, {
    pollingInterval: SERVICE_REFETCH_INTERVAL_MS,
  });

  const service_select = [
    {
      ip: "",
      port: 0,
      name: "all",
    },
    ...(services || []),
  ];
  let [searchParams, setSearchParams] = useSearchParams();
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
  useHotkeys('s', (e) => {
    let el = document.getElementById('search') as HTMLInputElement;
    el?.focus();
    el?.select();
    e.preventDefault()
  });
  return (
    <div>
      <input
        type="text"
        placeholder="regex"
        id="search"
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

  const { data: tickInfoData } = useGetTickInfoQuery(undefined, {
    pollingInterval: TICK_REFETCH_INTERVAL_MS,
  });

  // TODO: prevent having to work with default values here
  let startDate = "1970-01-01T00:00:00Z";
  let tickLength = 1000;

  if (tickInfoData) {
    startDate = tickInfoData.startDate;
    tickLength = tickInfoData.tickLength;
  }

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
  const currentTick = unixTimeToTick(new Date().valueOf().toString());

  function setToLastnTicks(n: number) {
    const startTick = (currentTick ?? 0) - n;
    const endTick = (currentTick ?? 0) + 1; // to be sure
    setTimeParam(startTick.toString(), START_FILTER_KEY);
    setTimeParam(endTick.toString(), END_FILTER_KEY);
  }

  return {
    unixTimeToTick,
    startDate,
    tickLength,
    setTimeParam,
    startTick,
    endTick,
    currentTick,
    setToLastnTicks,
  };
}

function StartDateSelection() {
  const { setTimeParam, startTick } = useMessyTimeStuff();

  return (
    <div>
      <input
        className="w-20"
        id="startdateselection"
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
        id="enddateselection"
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

function FirstDiff() {
  let params = useParams();
  let [searchParams, setSearchParams] = useSearchParams();
  const [firstFlow, setFirstFlow] = useState<string>(
    searchParams.get(FIRST_DIFF_KEY) ?? ""
  );

  function setFirstDiffFlow() {
    let textFilter = params.id;
    if (textFilter) {
      searchParams.set(FIRST_DIFF_KEY, textFilter);
      setFirstFlow(textFilter);
    } else {
      searchParams.delete(FIRST_DIFF_KEY);
      setFirstFlow("");
    }
    setSearchParams(searchParams);
  }

  useHotkeys("f", () => {
    setFirstDiffFlow();
  });

  return (
    <input
      type="text"
      className="md:w-72"
      placeholder="First Diff ID"
      readOnly
      value={firstFlow}
      onClick={(event) => setFirstDiffFlow()}
      onContextMenu={(event) => {
        searchParams.delete(FIRST_DIFF_KEY);
        setFirstFlow("");
        setSearchParams(searchParams);
        event.preventDefault();
      }}
    ></input>
  );
}

function SecondDiff() {
  let params = useParams();
  let [searchParams, setSearchParams] = useSearchParams();
  const [secondFlow, setSecondFlow] = useState<string>(
    searchParams.get(SECOND_DIFF_KEY) ?? ""
  );

  function setSecondDiffFlow() {
    let textFilter = params.id;
    if (textFilter) {
      searchParams.set(SECOND_DIFF_KEY, textFilter);
      setSecondFlow(textFilter);
    } else {
      searchParams.delete(SECOND_DIFF_KEY);
      setSecondFlow("");
    }
    setSearchParams(searchParams);
  }

  useHotkeys("g", () => {
    setSecondDiffFlow();
  });

  return (
    <input
      type="text"
      className="md:w-72"
      placeholder="Second Flow ID"
      readOnly
      value={secondFlow}
      onClick={(event) => setSecondDiffFlow()}
      onContextMenu={(event) => {
        searchParams.delete(SECOND_DIFF_KEY);
        setSecondFlow("");
        setSearchParams(searchParams);
        event.preventDefault();
      }}
    ></input>
  );
}

function Diff() {
  let params = useParams();

  let [searchParams] = useSearchParams();

  let navigate = useNavigate();

  function navigateToDiff() {
    navigate(`/diff/${params.id ?? ""}?${searchParams}`, { replace: true });
  }

  useHotkeys("d", () => {
    navigateToDiff();
  });

  return (
    <button
      className=" bg-amber-100 text-gray-800 rounded-md px-2 py-1"
      onClick={() => {
        navigateToDiff()
      }}
    >
      Diff
    </button>
  );
}

function SimilaritySlider() {
    let [searchParams, setSearchParams] = useSearchParams();
    // Initialize state to keep track of the value
    const [value, setValue] = useState(searchParams.get(SIMILARITY_FILTER_KEY) ?? 90); // You can set the initial value to whatever you prefer

    // Function to update the state based on input changes
    const handleChange = (event:any) => {
        setValue(event.target.value);
        searchParams.set(SIMILARITY_FILTER_KEY, event.target.value)
        setSearchParams(searchParams);
    };

    return (
        <div className="flex items-center">
            <div className="pr-3">
                <span>
                    Similarity:
                </span>
                <input
                    style={{paddingTop:0, paddingBottom:0}}
                    type="range"
                    min="0" // Set the minimum value of the range
                    max="100" // Set the maximum value of the range
                    value={value} // Bind the range input to the value in the state
                    onChange={handleChange} // Update the state when the range value changes
                />
            </div>
            <input
                className="w-20"
                type="number"
                min="0" // Set the minimum value for the number input
                max="100" // Set the maximum value for the number input
                value={value} // Bind the number input to the value in the state
                onChange={handleChange} // Update the state when the number value changes
            />
        </div>
    );
}

export function Header() {
  let [searchParams] = useSearchParams();
  const { setToLastnTicks, currentTick, setTimeParam } = useMessyTimeStuff();

  useHotkeys('a', () => setToLastnTicks(5));
  useHotkeys('c', () => {
    (document.getElementById("startdateselection") as HTMLInputElement).value = "";
    (document.getElementById("enddateselection") as HTMLInputElement).value = "";
    setTimeParam("", START_FILTER_KEY);
    setTimeParam("", END_FILTER_KEY);
  });

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
      <div>
        <button
          className=" bg-amber-100 text-gray-800 rounded-md px-2 py-1"
          onClick={() => setToLastnTicks(5)}
        >
          Last 5 ticks
        </button>
      </div>
      <Link to={`/corrie?${searchParams}`}>
        <div className="bg-blue-100 text-gray-800 rounded-md px-2 py-1">
          Graph view
        </div>
      </Link>
        <SimilaritySlider></SimilaritySlider>
      <div className="ml-auto mr-4" style={{ display: "flex" }}>
        <div className="mr-4">
          <FirstDiff />
        </div>
        <div className="mr-4">
          <SecondDiff />
        </div>
        <div className="mr-6">
          <Suspense>
            <Diff />
          </Suspense>
        </div>
        <div
          className="ml-auto"
          style={{
            display: "flex",
            justifyContent: "center",
            alignContent: "center",
            flexDirection: "column",
          }}
        >
          Current: {currentTick}
        </div>
      </div>
    </>
  );
}
