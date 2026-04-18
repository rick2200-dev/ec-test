/**
 * Shipment types mirror shipping-svc
 * (`backend/services/shipping`). The HTTP envelope serializes timestamps
 * as RFC3339 strings (see `shipmentResponse` helper).
 */

export type ShipmentStatus = "pending" | "ready_to_ship" | "shipped" | "delivered" | "cancelled";

export interface Shipment {
  id: string;
  seller_id: string;
  order_id: string;
  buyer_auth0_id: string;
  status: ShipmentStatus;
  carrier?: string;
  tracking_number?: string;
  note?: string;
  shipped_at?: string | null;
  delivered_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ShipmentListResponse {
  items: Shipment[];
  total: number;
  limit: number;
  offset: number;
}

export interface RegisterShipmentRequest {
  carrier: string;
  tracking_number: string;
  /** ISO8601 / RFC3339; omit to default to now on the server. */
  shipped_at?: string;
  note?: string;
}

export interface MarkDeliveredRequest {
  /** ISO8601 / RFC3339; omit to default to now on the server. */
  delivered_at?: string;
}
