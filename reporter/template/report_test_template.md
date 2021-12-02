total error log {{.Total}}, covered error log {{.Cov}}
{{- println }}
{{- range $path, $cov := .Details -}}
{{if $cov.Coverage }} 
path {{$path}} coverrd count {{$cov.Coverage.CovCount}} 
log level {{$cov.Pattern.Level}} signatures {{- $cov.Pattern.Signature}} 
coverage detail:
{{- range $addr, $count := $cov.Coverage.CovCountByLog}} 
file {{$addr}} cover count {{$count}} 
{{- end}}
{{- println }}
{{- else}} {{- end}}
{{- end}}
