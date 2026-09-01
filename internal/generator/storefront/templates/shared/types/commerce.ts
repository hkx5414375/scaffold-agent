export type CommerceOrderStatus =
  | "pending_payment"
  | "confirmed"
  | "fulfilling"
  | "fulfilled"
  | "return_requested"
  | "returned"
  | "cancelled";

export interface CommerceCartLine {
  product_id: string;
  sku: string;
  name: string;
  unit_minor: string;
  quantity: string;
  line_minor: string;
}

export interface CommerceCart {
  id?: string;
  currency: string;
  status: "active" | "checked_out";
  lines: CommerceCartLine[];
  subtotal_minor: string;
  version: string;
}

export interface CommercePayment {
  provider: string;
  provider_ref: string;
  status: "requires_action" | "succeeded" | "failed";
  amount_minor: string;
  refunded_minor: string;
  currency: string;
}

export interface CommerceOrder {
  id: string;
  status: CommerceOrderStatus;
  currency: string;
  subtotal_minor: string;
  discount_minor: string;
  total_minor: string;
  lines: CommerceCartLine[];
  payment: CommercePayment;
  fulfillment_ref?: string;
  return_reason?: string;
  version: string;
  created_at: string;
}

export interface CommerceOrderPage {
  items: CommerceOrder[];
  next_cursor?: string;
}
