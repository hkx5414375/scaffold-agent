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
