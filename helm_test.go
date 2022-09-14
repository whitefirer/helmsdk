package helmsdk

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	flatten "github.com/jeremywohl/flatten/v2"
	helmclient "github.com/whitefirer/helmsdk/helmclient"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"
)

const (
	defaultKubeConfig = ""
)

func TestGetDeployedReleases(t *testing.T) {
	opt := &helmclient.KubeConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        "default", // Change this to the namespace you wish to install the chart in.
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          true, // Change this to false if you don't want linting.
			DebugLog: func(format string, v ...interface{}) {
				// Change this to your own logger. Default is 'log.Printf(format, v...)'.
			},
		},
		KubeContext: "",
		KubeConfig:  []byte(defaultKubeConfig),
	}

	helmClient, err := helmclient.NewClientFromKubeConf(opt)
	if err != nil {
		panic(err)
	}
	// List all deployed releases.
	deployedReleases, err := helmClient.ListDeployedReleases()
	if err != nil {
		panic(err)
	}
	fmt.Printf("deployedReleases: %v\n", deployedReleases)
	for _, release := range deployedReleases {
		t.Logf("release Name: %v, Namespace: %v, Chart: %v, ChartVesion: %v,Version: %v\n", release.Name, release.Namespace, release.Chart.Name(), release.Chart.Metadata.Version, release.Version)
	}
}

func TestGetDeployedRelease(t *testing.T) {
	release, err := GetRelease(os.Stdout, "demo", "https://127.0.0.1", "docker-desktop", defaultKubeConfig, "wesing-service")
	if err != nil {
		panic(err)
	}
	t.Logf("Release Name: %v, Namespace: %v, Status: %v", release.Name, release.Namespace, release.Info.Status)
}

func TestGetDeployedReleasesByRestConfig(t *testing.T) {
	deployedReleases, err := GetReleaseList(os.Stdout, "https://127.0.0.1", "docker-desktop", defaultKubeConfig, "wesing-service")
	if err != nil {
		panic(err)
	}
	for _, release := range deployedReleases {
		t.Logf("release Name: %v, Namespace: %v, Chart: %v, ChartVesion: %v,Version: %v\n", release.Name, release.Namespace, release.Chart.Name(), release.Chart.Metadata.Version, release.Version)
	}
}

func TestGetDeployedReleaseByRestConfig(t *testing.T) {
	deployedRelease, err := GetRelease(os.Stdout, "demo", "https://127.0.0.1", "docker-desktop", defaultKubeConfig, "wesing-service")
	if err != nil {
		panic(err)
	}
	t.Logf("Release Name: %v, Status: %v", deployedRelease.Name, deployedRelease.Info.Status)
}

func TestGetDeployedReleaseValuesByRestConfig(t *testing.T) {
	_, restConfig, err := BuildK8sClientByKubeConfig(defaultKubeConfig, "https://127.0.0.1", "docker-desktop")
	if err != nil {
		panic(err)
	}
	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        "wesing-service", // Change this to the namespace you wish the client to operate in.
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
		panic(err)
	}

	// Get the values of a deployed release.
	values, err := helmClient.GetReleaseValues("demo", true)
	if err != nil {
		panic(err)
	}
	t.Logf("Release values: %v", values)
}

func TestGetDeployedReleaseValues(t *testing.T) {
	deployedReleaseValues, err := GetReleaseValues(os.Stdout, "demo", "https://127.0.0.1", "docker-desktop", defaultKubeConfig, "wesing-service", true)
	if err != nil {
		panic(err)
	}
	t.Logf("Release values:\n%v\n", deployedReleaseValues)
	jsonValues, err := json.Marshal(deployedReleaseValues)
	if err != nil {
		panic(err)
	}
	// yamlValues, err := yaml.Marshal(deployedReleaseValues)
	// if err != nil {
	// 	panic(err)
	// }
	// t.Logf("Release yaml:\n%s\n", string(yamlValues))
	dotValues, err := flatten.FlattenString(string(jsonValues), "", flatten.DotStyle)
	if err != nil {
		panic(err)
	}
	t.Logf("Release dotValues:\n%s\n", dotValues)

	flatValues, err := flatten.FlattenString(string(jsonValues), "", flatten.SeparatorStyle{Middle: "__"})
	if err != nil {
		panic(err)
	}
	t.Logf("Release flatValues:\n%s\n", flatValues)
	mapValue := map[string]interface{}{}
	err = json.Unmarshal([]byte(flatValues), &mapValue)
	if err != nil {
		panic(err)
	}
	t.Logf("Release mapValues:\n%v\n", mapValue)
}

func TestInstallOrUpgradeChart(t *testing.T) {
	// Define a private chart repository

	chartRepo := repo.Entry{
		Name:     "incubator",
		URL:      "https://charts.helm.sh/incubator",
		Username: "",
		Password: "",
		// Since helm 3.6.1 it is necessary to pass 'PassCredentialsAll = true'.
		PassCredentialsAll: true,
	}
	vals := map[string]string{
		"set": "image.tag=feature-timeout,autoscaling.enabled=true,autoscaling.minReplicas=3",
	}

	release, err := InstallOrUpgradeChart(os.Stdout, chartRepo, "demo", "service", "3.0.0", "https://127.0.0.1", "docker-desktop", defaultKubeConfig, "dev", vals)
	if err != nil {
		t.Logf("err msg: %v", err)
	}
	t.Logf("release Name: %v, Status: %v", release.Name, release.Info.Status)
}

func TestInstallOrUpgradeChart2(t *testing.T) {
	// Define a private chart repository

	chartRepo := repo.Entry{
		Name:     "incubator",
		URL:      "https://charts.helm.sh/incubator",
		Username: "",
		Password: "",
		// Since helm 3.6.1 it is necessary to pass 'PassCredentialsAll = true'.
		PassCredentialsAll: true,
	}
	vals := map[string]string{
		"set": "image.tag=chore-dc-security-scan,autoscaling.enabled=true,autoscaling.minReplicas=2",
	}

	release, err := InstallOrUpgradeChart(os.Stdout, chartRepo, "demo", "service", "3.0.0", "https://127.0.0.1", "docker-desktop", defaultKubeConfig, "wesing-service", vals)
	if err != nil {
		t.Logf("err msg: %v", err)
	}
	t.Logf("release Name: %v, Status: %v", release.Name, release.Info.Status)
}

func TestToYAML(t *testing.T) {
	// The TestParse does the hard part. We just verify that YAML formatting is
	// happening.
	testCases := []struct {
		str string
	}{
		{
			str: "name=value",
		},
		{
			str: "name1=value1,name2=value2",
		},
		{
			str: "name1=value1,name2=value2,",
		},
		{
			str: "outer.inner1=value1,outer.inner3=value3,outer.inner4=4",
		},
		{
			str: "leading_zeros=00009",
		},
		{
			str: "boolean=true",
		},
		{
			str: "name1={1021,902}",
		},
		{
			str: "nlist[0]=foo",
		},
		{
			str: "list[0].foo=bar",
		},
		{
			str: "list[0].foo=bar,list[0].hello=world",
		},
		{
			str: "name1.name2[0].foo=bar,name1.name2[1].foo=bar",
		},
		{
			str: "annotation.sidecar\\.istio\\.io/proxyCPU=100m",
		},
		{
			str: "autoscaling.enabled=true",
		},
		{
			str: `autoscaling\.enabled=true`,
		},
		{
			str: "image.tag=feature-timeout,autoscaling.enabled=true,autoscaling.minReplicas=3",
		},
		{
			str: "image.tag=chore-dc-security-scan,autoscaling.enabled=true,autoscaling.minReplicas=2",
		},
		{
			str: "extraLabels={failure-domain.beta.kubernetes.io/zone: us-west-1b,foo: bar}",
		},
	}
	for index, testCase := range testCases {
		out, err := strvals.ToYAML(testCase.str)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\ncase%d result:\n%s\n", index, out)
	}
}
