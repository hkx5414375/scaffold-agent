export type CustomerStatus = "active" | "suspended" | "closed";

export interface CustomerAccount {
  id: string;
  organization_id?: string;
  email: string;
  display_name: string;
  status: CustomerStatus;
  version: string;
  created_at: string;
  updated_at: string;
}

export interface CustomerEnvelope {
  customer: CustomerAccount;
}

export interface CustomerPasswordChanged extends CustomerEnvelope {
  reauthentication_required: true;
}
