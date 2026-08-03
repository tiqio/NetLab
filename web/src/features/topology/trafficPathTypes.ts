export interface TrafficObservation {
  fingerprint: string;
  resource_type?: string;
  resource_id?: string;
  interface_id: string;
  link_id?: string;
  network_object_link_id?: string;
  direction: string;
  first_seen: string;
  last_seen: string;
  count: number;
  bytes: number;
}
