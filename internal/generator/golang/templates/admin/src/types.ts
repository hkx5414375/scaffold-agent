export interface Principal {
  user_id: string;
  email: string;
  role: string;
  method: "session" | "token";
}

export interface PrincipalResponse {
  principal: Principal;
}

{{if .Business}}export interface BusinessEntity {
  id: string;
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
