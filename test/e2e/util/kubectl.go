// https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/test/e2e/framework/exec/kubectl.go
package util

import (
	"fmt"
	"os/exec"
	"strings"

	"k8s.io/klog/v2"
)

// KubectlApply executes "kubectl apply" given a list of arguments.
func KubectlApply(kubeconfigPath, namespace string, args []string) error {
	args = append([]string{
		"apply",
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--namespace=%s", namespace),
	}, args...)

	_, err := Kubectl(args)
	return err
}

// HelmInstall executes "helm install" given a list of arguments.
func HelmInstall(kubeconfigPath, namespace string, args []string) error {
	args = append([]string{
		"install",
		"eraser-e2e-test",
		"--wait",
		"--debug",
		"--create-namespace",
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--namespace=%s", namespace),
	}, args...)

	_, err := Helm(args)
	return err
}

// HelmUninstall executes "helm uninstall" given a list of arguments.
func HelmUninstall(kubeconfigPath, namespace string, args []string) error {
	args = append([]string{
		"uninstall",
		"eraser-e2e-test",
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--namespace=%s", namespace),
	}, args...)

	_, err := Helm(args)
	return err
}

// KubectlDelete executes "kubectl delete" given a list of arguments.
func KubectlDelete(kubeconfigPath, namespace string, args []string) error {
	args = append([]string{
		"delete",
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--namespace=%s", namespace),
	}, args...)

	_, err := Kubectl(args)
	return err
}

// KubectlExec executes "kubectl exec" given a list of arguments.
func KubectlExec(kubeconfigPath, podName, namespace string, args []string) (string, error) {
	args = append([]string{
		"exec",
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--namespace=%s", namespace),
		"--request-timeout=5s",
		podName,
		"--",
	}, args...)

	return Kubectl(args)
}

// KubectlLogs executes "kubectl logs" given a list of arguments.
func KubectlLogs(kubeconfigPath, podName, containerName, namespace string) (string, error) {
	args := []string{
		"logs",
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--namespace=%s", namespace),
		podName,
	}

	if containerName != "" {
		args = append(args, fmt.Sprintf("-c=%s", containerName))
	}

	return Kubectl(args)
}

// KubectlDescribe executes "kubectl describe" given a list of arguments.
func KubectlDescribe(kubeconfigPath, podName, namespace string) (string, error) {
	args := []string{
		"describe",
		"pod",
		podName,
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--namespace=%s", namespace),
	}
	return Kubectl(args)
}

// KubectlDescribeImagejob executes "kubectl describe imagejob".
func KubectlGet(kubeconfigPath string, otherArgs ...string) (string, error) {
	args := []string{
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		"get",
	}
	args = append(args, otherArgs...)

	return Kubectl(args)
}

func Kubectl(args []string) (string, error) {
	klog.Infof("kubectl %s", strings.Join(args, " "))

	cmd := exec.Command("kubectl", args...)

	stdoutStderr, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdoutStderr))
	if err != nil {
		err = fmt.Errorf("%w: %s", err, output)
	}

	return output, err
}

func Helm(args []string) (string, error) {
	klog.Infof("helm %s", strings.Join(args, " "))

	cmd := exec.Command("helm", args...)

	stdoutStderr, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdoutStderr))
	if err != nil {
		err = fmt.Errorf("%w: %s", err, output)
	}

	return output, err
}

func MakeDeploy() (string, error) {
	cmd := exec.Command("make", "deploy")

	stdoutStderr, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdoutStderr))
	if err != nil {
		err = fmt.Errorf("%w: %s", err, output)
	}

	return output, err
}
