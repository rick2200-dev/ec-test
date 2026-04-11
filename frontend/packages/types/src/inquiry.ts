/**
 * Inquiry types mirror the shapes returned by the inquiry service
 * (`backend/services/inquiry`). Both buyer and seller apps consume the
 * same JSON, so the types live here rather than being duplicated.
 */

/** Inquiry thread (1 thread per buyer × seller × SKU). */
export interface Inquiry {
  id: string;
  tenant_id: string;
  buyer_auth0_id: string;
  seller_id: string;
  sku_id: string;
  product_name: string;
  sku_code: string;
  subject: string;
  status: "open" | "closed";
  last_message_at: string;
  created_at: string;
  updated_at: string;
  unread_count?: number;
}

/** A single message inside an inquiry thread. */
export interface InquiryMessage {
  id: string;
  tenant_id: string;
  inquiry_id: string;
  sender_type: "buyer" | "seller";
  sender_id: string;
  body: string;
  read_at?: string | null;
  created_at: string;
}

/** Inquiry thread with its full message history. */
export interface InquiryWithMessages extends Inquiry {
  messages: InquiryMessage[];
}

/** Paginated inquiry list response. */
export interface InquiryListResponse {
  items: Inquiry[];
  total: number;
  limit: number;
  offset: number;
}
