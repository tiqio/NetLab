export interface TrafficObservation {
  fingerprint: string;
  interface_id: string;
  link_id?: string;
  direction: string;
  first_seen: string;
  last_seen: string;
  count: number;
  bytes: number;
}
