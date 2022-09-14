package helmsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"

	helmclient "github.com/whitefirer/helmsdk/helmclient"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

func InstallOrUpgradeChart(out io.Writer, chartRepo repo.Entry, releaseName string, chartName string, chartVersion string, apiServer string, apiServerName string, kubeConfig string, namespace string, args map[string]string) (*release.Release, error) {
	setStr, ok := args["set"]
	if !ok {
		return nil, fmt.Errorf("pramas fomrt error. miss 'set'")
	}
	helmClient, err := GetHelmClient(apiServer, apiServerName, kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	// Add a chart-repository to the client.
	if err := helmClient.AddOrUpdateChartRepo(chartRepo); err != nil {
		return nil, err
	}

	yamlStr, err := strvals.ToYAML(setStr)
	if err != nil {
		return nil, fmt.Errorf("pramas fomrt error")
	}

	// Define the chart to be installed
	chartSpec := helmclient.ChartSpec{
		ReleaseName: releaseName,
		ChartName:   fmt.Sprintf("%s/%s", chartRepo.Name, chartName),
		Version:     chartVersion,
		Namespace:   namespace,
		ValuesYaml:  yamlStr,
		UpgradeCRDs: true,
		ReuseValues: true,
		Wait:        true,
		Timeout:     600 * time.Second,
		MaxHistory:  20,
	}

	// Install a chart release.
	// Note that helmclient.Options.Namespace should ideally match the namespace in chartSpec.Namespace.
	release, err := helmClient.InstallOrUpgradeChart(context.Background(), &chartSpec, nil)
	if err != nil {
		return nil, err
	}

	return release, nil
}

func RollBackRelease(out io.Writer, chartRepoName string, releaseName string, chartName string, apiServer string, apiServerName string, kubeConfig string, namespace string) error {
	chartSpec := helmclient.ChartSpec{
		ReleaseName: releaseName,
		ChartName:   fmt.Sprintf("%s/%s", chartRepoName, chartName),
		Namespace:   namespace,
		UpgradeCRDs: true,
		Wait:        true,
		Timeout:     600 * time.Second,
		MaxHistory:  20,
	}

	helmClient, err := GetHelmClient(apiServer, apiServerName, kubeConfig, namespace)
	if err != nil {
		return err
	}

	// Rollback to the previous version of the release.
	if err := helmClient.RollbackRelease(&chartSpec); err != nil {
		return err
	}

	return nil
}

func GetReleaseList(out io.Writer, apiServer string, apiServerName string, kubeConfig string, namespace string) ([]*release.Release, error) {
	helmClient, err := GetHelmClient(apiServer, apiServerName, kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	// List all deployed releases.
	releases, err := helmClient.ListDeployedReleases()
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func GetRelease(out io.Writer, releaseName string, apiServer string, apiServerName string, kubeConfig string, namespace string) (*release.Release, error) {
	helmClient, err := GetHelmClient(apiServer, apiServerName, kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	// Get the release
	release, err := helmClient.GetRelease(releaseName)
	if err != nil {
		return nil, err
	}

	return release, nil
}

func GetReleaseValues(out io.Writer, releaseName string, apiServer string, apiServerName string, kubeConfig string, namespace string, allValues bool) (map[string]interface{}, error) {
	helmClient, err := GetHelmClient(apiServer, apiServerName, kubeConfig, namespace)
	if err != nil {
		return nil, err
	}

	// Get the release values
	values, err := helmClient.GetReleaseValues(releaseName, allValues)
	if err != nil {
		return nil, err
	}

	return values, nil
}

func GetHelmClient(apiServer string, apiServerName string, kubeConfig string, namespace string) (helmclient.Client, error) {
	_, restConfig, err := BuildK8sClientByKubeConfig(kubeConfig, apiServer, apiServerName)
	if err != nil {
		return nil, err
	}
	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        namespace, // Change this to the namespace you wish the client to operate in.
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          true, // Change this to false if you don't want linting.
			DebugLog: func(format string, v ...interface{}) {
				// Change this to your own logger. Default is 'log.Printf(format, v...)'.
			},
		},
		RestConfig: restConfig,
	}

	helmClient, err := helmclient.NewClientFromRestConf(opt)
	if err != nil {
		return nil, err
	}

	return helmClient, nil
}

const (
	// High enough QPS to fit all expected use cases.
	defaultQPS = 1e6
	// High enough Burst to fit all expected use cases.
	defaultBurst = 1e6
	// full resyc cache resource time
	// defaultResyncPeriod = 30 * time.Second
)

func BuildK8sClient(server string, serverName string, configV1 clientcmdapiv1.Config) (*kubernetes.Clientset, *rest.Config, error) {
	configObject, err := clientcmdlatest.Scheme.ConvertToVersion(&configV1, clientcmdapi.SchemeGroupVersion)
	if err != nil {
		return nil, nil, err
	}
	configInternal := configObject.(*clientcmdapi.Config)

	clientConfig, err := clientcmd.NewDefaultClientConfig(*configInternal,
		&clientcmd.ConfigOverrides{
			ClusterDefaults: clientcmdapi.Cluster{
				Server:        server,
				TLSServerName: serverName,
			},
		}).ClientConfig()

	if err != nil {
		return nil, nil, err
	}

	clientConfig.QPS = defaultQPS
	clientConfig.Burst = defaultBurst
	clientConfig.Host = server
	clientConfig.TLSClientConfig.ServerName = serverName

	clientSet, err := kubernetes.NewForConfig(clientConfig)

	if err != nil {
		return nil, nil, err
	}

	return clientSet, clientConfig, nil
}

func BuildK8sClientByKubeConfig(kubeConfig string, apiServer string, apiServerName string) (*kubernetes.Clientset, *rest.Config, error) {
	k8sContext := clientcmdapiv1.Config{}

	if kubeConfig == "" {
		clientConfig, err := clientcmd.BuildConfigFromFlags("", fmt.Sprintf("%s/.kube/config", os.Getenv("HOME")))
		if err != nil {
			return nil, nil, err
		}
		clientSet, err := kubernetes.NewForConfig(clientConfig)
		if err != nil {
			return nil, nil, err
		}
		return clientSet, clientConfig, nil
	} else {
		data, err := yaml2.ToJSON([]byte(kubeConfig))
		if err != nil {
			return nil, nil, err
		}
		err = json.Unmarshal(data, &k8sContext)
		if err != nil {
			return nil, nil, err
		}
	}

	return BuildK8sClient(apiServer, apiServerName, k8sContext)
}
