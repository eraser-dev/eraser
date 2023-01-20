package main

var replacements = map[string]string{
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_CONTAINER_RESOURCES: ""`: `{{- toYaml .Values.deploy.resources | nindent 10 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_NODESELECTOR: ""`:        `{{- toYaml .Values.deploy.nodeSelector | nindent 8 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_TOLERATIONS: ""`:         `{{- toYaml .Values.deploy.tolerations | nindent 8 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_AFFINITY: ""`:            `{{- toYaml .Values.deploy.affinity | nindent 8 }}`,
	`- HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_ADDITIONAL_ARGS`:       `{{- if .Values.deploy.additionalArgs }}{{- range .Values.deploy.additionalArgs }}{{ nindent 8 "- " }}{{ . }}{{- end -}}{{ end }}`,
	`HELMSUBST_CONTROLLER_MANAGER_CONFIG_YAML`:                        `{{- toYaml .Values.runtimeConfig | nindent 4 }}`,

	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_PULL_SECRETS: ""`: `{{- if .Values.runtimeConfig.manager.pullSecrets }}
      imagePullSecrets:
        {{- range .Values.runtimeConfig.manager.pullSecrets }}
        - name: {{ . }}
        {{- end }}
      {{- end }}`,

	`- HELMSUBST_VOLUME: ""`: `- name: helm-config
        configMap:
          name: "eraser-manager-config-{{ toYaml .Values.runtimeConfig | sha256sum | trunc 7 }}"
      `,

	`- HELMSUBST_VOLUMEMOUNT: ""`: `- name: helm-config
          mountPath: /helm_controller_manager_config.yaml
          subPath: controller_manager_config.yaml
      `,
	`HELMSUBST_CONFIG_FILE_FLAG`: `--config=helm_controller_manager_config.yaml`,
}
