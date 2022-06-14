import { useAtom } from "jotai";
import { atomWithStorage } from "jotai/utils";

const tickLengthInMs = atomWithStorage<number>("tickLengthInMs", 2 * 60 * 1000);
const ctfStartTime = atomWithStorage<string>(
  "ctfStartTime2",
  "2022-07-16T09:00+03:00"
);

// Abstraction so we can maybe get this data from the server in the future
export const useCTF = function () {
  const [tickLength, updateTickLength] = useAtom(tickLengthInMs);
  const [startDate, updateStartDate] = useAtom(ctfStartTime);
  console.log(startDate, tickLength);

  return { tickLength, startDate };
};

function TickLengthSettings() {
  const [tickLength, updateTickLength] = useAtom(tickLengthInMs);
  return (
    <div>
      <label>Length of a tick in ms: </label>
      <input
        value={tickLength}
        type="number"
        onChange={(ev) => {
          updateTickLength(parseInt(ev.target.value));
        }}
      ></input>
    </div>
  );
}

function CTFStartSettings() {
  const [startDate, updateStartDate] = useAtom(ctfStartTime);
  return (
    <div>
      <label>Start time of the CTF: </label>
      <input
        value={startDate}
        type="datetime-local"
        onChange={(ev) => {
          updateStartDate(ev.target.value);
        }}
      ></input>
    </div>
  );
}

export function Home() {
  return (
    <div className="p-4">
      <h1 className="text-3xl font-bold pt-2 pb-4">Welcome to 🌷</h1>
      <div>
        <h2 className="text-lg">Settings</h2>
        <div>
          <TickLengthSettings></TickLengthSettings>
          <CTFStartSettings></CTFStartSettings>
        </div>
      </div>
    </div>
  );
}
