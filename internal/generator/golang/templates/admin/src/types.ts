export interface Principal {
  user_id: string;
  email: string;
  role: string;
  method: "session" | "token";
  organization_id?: string;
}

export interface PrincipalResponse {
  principal: Principal;
}
{{- if .Tenancy}}

export interface Organization {
  id: string;
  name: string;
  role: string;
  created_at: string;
{{- if .TenancyLifecycle}}
  owner_user_id: string;
  is_owner: boolean;
  status: "active" | "inactive";
  updated_at: string;
  deactivated_at?: string;
{{- end}}
}

export interface OrganizationPage {
  items: Organization[];
}
{{- if .TenancyMembers}}

export interface OrganizationMember {
  organization_id: string;
  user_id: string;
  email: string;
  role: "admin" | "user";
  joined_at: string;
{{- if .TenancyLifecycle}}
  is_owner: boolean;
{{- end}}
}

export interface OrganizationMemberPage {
  items: OrganizationMember[];
}

export interface OrganizationInvitation {
  id: string;
  organization_id: string;
  email: string;
  role: "admin" | "user";
  acceptance_token: string;
  expires_at: string;
  created_at: string;
}
{{- end}}
{{- end}}
{{- if .Files}}

export interface FileAsset {
  id: string;
  organization_id?: string;
  name: string;
  media_type: string;
  size: number;
  sha256: string;
  created_at: string;
}

export interface FileAssetPage {
  items: FileAsset[];
  next_cursor?: string;
}
{{- end}}
{{- if .JobAdmin}}

export interface JobItem {
  id: string;
  organization_id?: string;
  type: string;
  status: "queued" | "running" | "succeeded" | "retry" | "dead";
  priority: number;
  attempts: number;
  max_attempts: number;
  available_at: string;
  lease_owner?: string;
  lease_until?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface JobPage {
  items: JobItem[];
  next_cursor?: string;
}
{{- end}}
{{- if .Business}}

export interface BusinessEntity {
  id: string;
{{- if .Tenancy}}
  organization_id: string;
{{- end}}
{{- range .Business.Fields}}
  {{.Name}}{{if not .Required}}?{{end}}: {{.TypeScriptType}};
{{- end}}
  version: string;
  created_at: string;
  updated_at: string;
}

export interface BusinessWrite {
{{- range .Business.Fields}}
  {{.Name}}{{if not .Required}}?{{end}}: {{.TypeScriptType}};
{{- end}}
}

export interface BusinessPage {
  items: BusinessEntity[];
  next_cursor?: string;
}

export function emptyBusinessWrite(): BusinessWrite {
  return {
{{- range .Business.Fields}}
    {{.Name}}: {{if .Required}}{{.TypeScriptDefault}}{{else}}undefined{{end}},
{{- end}}
  };
}
{{- end}}
