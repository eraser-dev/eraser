package main

var replacements = map[string]string{
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_CONTAINER_RESOURCES: ""`: `{{- toYaml .Values.controllerManager.resources | nindent 10 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_NODESELECTOR: ""`:        `{{- toYaml .Values.controllerManager.nodeSelector | nindent 8 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_TOLERATIONS: ""`:         `{{- toYaml .Values.controllerManager.tolerations | nindent 8 }}`,
	`HELMSUBST_DEPLOYMENT_CONTROLLER_MANAGER_AFFINITY: ""`:            `{{- toYaml .Values.controllerManager.affinity | nindent 8 }}`,
}
