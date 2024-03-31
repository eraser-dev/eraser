package main

var replacements = map[string]string{
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_CONTAINER_RESOURCES: ""`: `{{- toYaml .Values.deploy.resources | nindent 10 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_NODESELECTOR: ""`:        `{{- toYaml .Values.deploy.nodeSelector | nindent 8 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_TOLERATIONS: ""`:         `{{- toYaml .Values.deploy.tolerations | nindent 8 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_AFFINITY: ""`:            `{{- toYaml .Values.deploy.affinity | nindent 8 }}`,
	`- HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_ADDITIONAL_ARGS`:       `{{- if .Values.deploy.additionalArgs }}{{- range .Values.deploy.additionalArgs }}{{ nindent 8 "- " }}{{ . }}{{- end -}}{{ end }}`,
	`HELMSUBST_CONTROLLER_MANAGER_CONFIG_YAML`:                        `{{- toYaml .Values.runtimeConfig | nindent 4 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_ADDITIONALPODLABELS: ""`: `{{- if .Values.deploy.additionalPodLabels }}{{- toYaml .Values.deploy.additionalPodLabels | nindent 8 }}{{end}}`,

	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_PULL_SECRETS: ""`: `{{- if .Values.runtimeConfig.manager.pullSecrets }}
      imagePullSecrets:
        {{- range .Values.runtimeConfig.manager.pullSecrets }}
        - name: {{ . }}
        {{- end }}
      {{- end }}`,
}
