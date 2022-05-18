import { useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import { api, FullFlow } from "../api";

export function FlowView() {
  const params = useParams();
  const [flow, setFlow] = useState<FullFlow>();

  const id = params.id;

  useEffect(() => {
    const fetchData = async () => {
      if (id === undefined) {
        return;
      }
      const data = await api.getFlow(id);
      setFlow(data);
    };
    fetchData().catch(console.error);
  }, [id]);

  return <h1>Flowview {JSON.stringify(flow)}</h1>;
}
