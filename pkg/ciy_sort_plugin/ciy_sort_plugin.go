package ciy_sort_plugin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

const (
	Name                         = "CiySortPlugin"
	clusterNodeDurationInMinutes = 10
	responseTimeoutInSecond      = 2
	ciyWeight                    = 0.6
	metricsWeight                = 0.4
	clusterNodeSurvivalApiPath   = "%s/api/v1/node_survival_chance/%s/%d"
	clusterAbruptShutdownApiPath = "%s/api/v1/abrupt_disconnects/%s"
)

type CiySortPlugin struct {
	clusterAccessURL string
	kubeClient       *kubernetes.Clientset
	metricsClient    *versioned.Clientset
	currentNodeMap   map[string]v1.Node
}

func NewCiySortPlugin() *CiySortPlugin {
	kubeClient, metricsClient := getKubernetesClient()
	clusterAccessURL := os.Getenv("CLUSTER_ACCESS_URL")

	return &CiySortPlugin{
		kubeClient:       kubeClient,
		currentNodeMap:   getNodeList(kubeClient),
		metricsClient:    metricsClient,
		clusterAccessURL: clusterAccessURL,
	}
}

func (ciy *CiySortPlugin) Name() string {
	return Name
}
func New(obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return NewCiySortPlugin(), nil
}

func getNodeList(kubeClient *kubernetes.Clientset) map[string]v1.Node {
	currentNodeList, err := kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	currentNodeMap := make(map[string]v1.Node)
	for _, node := range currentNodeList.Items {
		currentNodeMap[node.Name] = node
	}
	return currentNodeMap
}

func getKubernetesClient() (*kubernetes.Clientset, *versioned.Clientset) {
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	// Create a Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	metricsClient, metrics_err := versioned.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	if metrics_err != nil {
		panic(metrics_err)
	}
	return clientset, metricsClient

}
func runTimedHttpRequest(ctx context.Context, method string, url string) (string, error) {
	context, cancel := context.WithTimeout(ctx, responseTimeoutInSecond*time.Second)
	defer cancel() // Ensure resources are cleaned up

	req, err := http.NewRequestWithContext(context, method, url, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{}

	// Send the request using the client
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error readingbody:", err)
		return "", err
	}
	return string(respBody), nil
}

func (ciy *CiySortPlugin) IsNodePersistent(nodeName string) (bool, error) {
	nodeValue, ok := ciy.currentNodeMap[nodeName]
	if !ok {
		ciy.currentNodeMap = getNodeList(ciy.kubeClient)
		nodeValue, ok = ciy.currentNodeMap[nodeName]
		if !ok {
			return false, fmt.Errorf("node %s not found", nodeName)
		}
	}
	_, exists := nodeValue.ObjectMeta.Labels["ciy.persistent_node"]
	return exists, nil

}

func (ciy *CiySortPlugin) GetCiyScore(ctx context.Context, nodeName string, isNodePersistent bool) (float64, *framework.Status) {
	if isNodePersistent {
		return 0.5, nil // return median score
	}
	abruptionResp, err := runTimedHttpRequest(ctx, http.MethodGet, fmt.Sprintf(clusterAbruptShutdownApiPath, ciy.clusterAccessURL, nodeName))
	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}

	abruptionChance, err := strconv.ParseFloat(abruptionResp, 64)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}

	nodeSurvivalChanceResp, err := runTimedHttpRequest(ctx, http.MethodGet, fmt.Sprintf(clusterNodeSurvivalApiPath, ciy.clusterAccessURL, nodeName, clusterNodeDurationInMinutes))
	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}
	survivalChance, err := strconv.ParseFloat(nodeSurvivalChanceResp, 64)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}
	return abruptionChance * survivalChance, nil
}

func (ciy *CiySortPlugin) getMetricsServerScore(ctx context.Context, nodeName string) (float64, *framework.Status) {
	nodeDetails, err := ciy.kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	nodeMetrics, metrics_err := ciy.metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil || metrics_err != nil {
		fmt.Printf("Error fetching metrics for node %s: %v\n", nodeName, err)
		return 0.5, nil
	}
	allocatableCPU := nodeDetails.Status.Allocatable["cpu"]
	allocatableCPUMilliCores := allocatableCPU.MilliValue()

	// Calculate CPU usage percentage
	cpuUsageMilliCores := nodeMetrics.Usage.Cpu().MilliValue()
	cpuUsagePercentage := 1.0 - float64(cpuUsageMilliCores)/float64(allocatableCPUMilliCores)

	allocatableMemory := nodeDetails.Status.Allocatable["memory"]
	allocatableMemoryBytes := allocatableMemory.Value()

	memoryUsageBytes := nodeMetrics.Usage.Memory().Value()
	memoryUsagePercentage := 1.0 - float64(memoryUsageBytes)/float64(allocatableMemoryBytes)

	return cpuUsagePercentage * memoryUsagePercentage, nil
}

func (ciy *CiySortPlugin) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	isNodePersistent, err := ciy.IsNodePersistent(nodeName)
	if err != nil {
		fmt.Printf("Error fetching metrics for node %s: %v, score is 0\n", nodeName, err)
		return 0, framework.NewStatus(framework.Error, err.Error())
	}

	if (p.ObjectMeta.Namespace == "kube-system" || p.ObjectMeta.Namespace == "cloud-iy") && p.ObjectMeta.OwnerReferences[0].Kind != "DaemonSet" {
		if isNodePersistent {
			metricsScore, _ := ciy.getMetricsServerScore(ctx, nodeName)
			fmt.Printf("Score for %s is %d\n", nodeName, int64(metricsScore*100.0))
			return int64(metricsScore * 100.0), nil
		} else {
			return 0, nil
		}
	}
	ciyScore, ciyErr := ciy.GetCiyScore(ctx, nodeName, isNodePersistent)
	if ciyErr != nil {
		fmt.Printf("Error fetching ciy score for node %s: %v, score is 0\n", nodeName, err)
		return 0, ciyErr
	}
	metricsScore, _ := ciy.getMetricsServerScore(ctx, nodeName)
	fmt.Printf("Node: %s, ciy-score: %f, metrics-score: %f, total-score: %d\n", nodeName, ciyScore, metricsScore, int64((ciyScore*ciyWeight+metricsScore*metricsWeight)*100.0))
	return int64((ciyScore*ciyWeight + metricsScore*metricsWeight) * 100.0), nil
}

func (ciy *CiySortPlugin) ScoreExtensions() framework.ScoreExtensions {
	return ciy
}

func (ciy *CiySortPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	return nil
}
