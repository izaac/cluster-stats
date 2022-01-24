package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	cs "github.com/izaac/cluster-stats/v2"
)

func main() {
	var kubeconfig *string
	var c cs.Conf
	configPath := c.GetConf().Kubeconfigpath
	configNames, err := cs.GetKubeConfigsList(configPath)
	if err != nil {
		panic(err)
	}

	for i, configName := range configNames {
		fmt.Println(filepath.Join(configPath, configName))
		kubeconfig = flag.String(fmt.Sprintf("kubeconfig%d", i), filepath.Join(configPath, configName), "absolute path to kubeconfig")
		flag.Parse()
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err)
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}
		options := v1.ListOptions{}
		ctx := context.TODO()
		namespaceList, err := clientset.CoreV1().Namespaces().List(ctx, options)
		if err != nil {
			fmt.Print(err.Error())
		}

		fmt.Println()
		totalCRBs, err := cs.GetTotalCRBs(clientset, &ctx, &options)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("- ClusterRoleBindings Total: %d\n", totalCRBs)

		totalSecrets, err := cs.GetSecretsStats(clientset, &ctx, &options, namespaceList)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("- Secrets Total: %d\n", totalSecrets)

		roleBindingsTotal, err := cs.GetTotalRBs(clientset, &ctx, &options, namespaceList)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("- RoleBindings Total: %d\n", roleBindingsTotal)

		largestCMSize, largestCMName, largestCMNamespace, mapOfLargestCMs, err := cs.GetConfigMapStats(clientset, &ctx, &options, namespaceList)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("- Largest ConfigMap Size: %d\n", largestCMSize)
		fmt.Printf("- Largest ConfigMap Name: %s\n", largestCMName)
		fmt.Printf("- Namespace with largest ConfigMap : %s\n", largestCMNamespace)
		fmt.Println()
		fmt.Println("=== ConfigMaps Name and Size ===")
		cs.PrintMap(mapOfLargestCMs)
		kubeconfig = nil
		fmt.Println()
	}
}