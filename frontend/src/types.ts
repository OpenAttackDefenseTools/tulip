export interface Flow {
  _id: Id;
  src_port: number;
  dst_port: number;
  src_ip: string;
  dst_ip: string;
  time: number;
  duration: number;
  // TODO: Get this from backend instead of hacky workaround
  service_tag: string;
  num_packets: number;
  parent_id: Id;
  child_id: Id;
  tags: string[];
  suricata: number[];
  filename: string;
}

export interface TickInfo {
  startDate: string;
  tickLength: number;
}

export interface FullFlow extends Flow {
  signatures: Signature[];
  flow: FlowData[];
}

export interface Id {
  $oid: string;
}

export interface FlowData {
  from: string;
  data: string;
  b64: string;
  time: number;
}

export interface Signature {
  id: number;
  msg: string;
  action: string;
}

// TODO: pagination WTF
export interface FlowsQuery {
  // Text filter
  "flow.data"?: string;
  // Service filter
  // TODO: Why not use service name here?
  service: string;
  dst_ip?: string;
  dst_port?: number;
  from_time?: string;
  to_time?: string;
  includeTags: string[];
  excludeTags: string[];
}

export type Service = {
  ip: string;
  port: number;
  name: string;
};
