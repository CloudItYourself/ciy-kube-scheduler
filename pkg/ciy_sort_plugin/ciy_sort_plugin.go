package ciy_sort_plugin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	percentageConvertionUnit     = 10000
	clusterNodeDurationInMinutes = 10
	responseTimeoutInSecond      = 2
	clusterNodeSurvivalApiPath   = "/api/v1/node_survival_chance/%s/%d"
	clusterAbruptShutdownApiPath = "/api/v1/abrupt_disconnects/%s"
)

type CiySortPlugin struct {
}

func runTimedHttpRequest(ctx context.Context, method string, url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), responseTimeoutInSecond*time.Second)
	defer cancel() // Ensure resources are cleaned up

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
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
func (n *CiySortPlugin) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	resp, err := runTimedHttpRequest(ctx, http.MethodGet, fmt.Sprintf(clusterAbruptShutdownApiPath, nodeName))
	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}
	float_val, err := strconv.ParseFloat(resp, 64)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}
	return int64(float_val) * percentageConvertionUnit, nil
}

func (n *CiySortPlugin) ScoreExtensions() framework.ScoreExtensions {
	return n
}

func (n *CiySortPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {

	return nil
}
