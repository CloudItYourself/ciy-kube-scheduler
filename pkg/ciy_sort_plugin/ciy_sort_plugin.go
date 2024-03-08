package ciy_sort_plugin

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	clusterNodeDurationInMinutes = 10
	clusterNodeSurvivalApiPath   = "/api/v1/node_survival_chance/{node_name}/{duration}"
	clusterAbruptShutdownApiPath = "/api/v1/abrupt_disconnects/{node_name}"
)

type CiySortPlugin struct {
}

func (n *CiySortPlugin) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	//klog.Infof("[NetworkTraffic] node '%s' bandwidth: %s", nodeName, nodeBandwidth.Value)
	return 100, nil
}

func (n *CiySortPlugin) ScoreExtensions() framework.ScoreExtensions {
	return n
}

func (n *CiySortPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {

	return nil
}
