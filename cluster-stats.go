package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type conf struct {
	Kubeconfigpath     string `yaml:"kubeconfigpath"`
	Configmaplargesize int    `yaml:"configmaplargesize"`
}

func (c *conf) getConf() *conf {

	yamlFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		fmt.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Print(err.Error())
	}
	return c
}

func printMap(m map[string]int) {
	var maxLenKey int
	for k := range m {
		if len(k) > maxLenKey {
			maxLenKey = len(k)
		}
	}
	type kv struct {
		Key   string
		Value int
	}

	var ss []kv
	for k, v := range m {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	for _, kv := range ss {
		fmt.Println(kv.Key + ": " + strings.Repeat(" ", maxLenKey-len(kv.Key)) + fmt.Sprint(kv.Value))
	}
}

func getKubeConfigsList(path string) ([]string, error) {
	items, err := ioutil.ReadDir(path)
	var configs []string
	if err != nil {
		fmt.Print(err.Error())
	}
	for _, item := range items {
		if item.IsDir() {
			subitems, _ := ioutil.ReadDir(item.Name())
			for _, subitem := range subitems {
				if !subitem.IsDir() {
					configs = append(configs, item.Name())
				}
			}
		} else {
			configs = append(configs, item.Name())
		}
	}
	return configs, err
}

func getTotalCRBs(client *kubernetes.Clientset, ctx *context.Context, opts *v1.ListOptions) (int, error) {
	crblist, err := client.RbacV1().ClusterRoleBindings().List(*ctx, *opts)
	total := 0

	if err != nil {
		fmt.Print(err.Error())
	}
	total = len(crblist.Items)
	return total, err
}

func getTotalRBs(client *kubernetes.Clientset, ctx *context.Context, opts *v1.ListOptions, nsList *apiv1.NamespaceList) (int, error) {
	roleBindingsTotal := 0
	err := error(nil)
	for _, ns := range nsList.Items {
		rbs, err := client.RbacV1().RoleBindings(ns.Name).List(*ctx, *opts)
		if err != nil {
			fmt.Print(err.Error())
		}
		roleBindingsLength := len(rbs.Items)
		roleBindingsTotal = roleBindingsTotal + roleBindingsLength
	}

	return roleBindingsTotal, err
}

func getSecretsStats(client *kubernetes.Clientset, ctx *context.Context, opts *v1.ListOptions, nsList *apiv1.NamespaceList) (int, error) {
	secretsTotal := 0
	err := error(nil)
	for _, ns := range nsList.Items {
		secrets, err := client.CoreV1().Secrets(ns.Name).List(*ctx, *opts)
		if err != nil {
			fmt.Print(err.Error())
		}
		secretsLength := len(secrets.Items)
		secretsTotal = secretsTotal + secretsLength
	}

	return secretsTotal, err
}

func getConfigMapStats(client *kubernetes.Clientset, ctx *context.Context, opts *v1.ListOptions, nsList *apiv1.NamespaceList) (int, string, string, map[string]int, error) {
	var c conf
	largestCMSize := 0
	largestCMName := ""
	largestCMNamespace := ""
	largeSizeOfCM := c.getConf().Configmaplargesize
	mapOfLargestCMs := map[string]int{}

	err := error(nil)

	for _, ns := range nsList.Items {
		cms, err := client.CoreV1().ConfigMaps(ns.Name).List(*ctx, *opts)
		if err != nil {
			fmt.Print(err.Error())
		}
		listOfCms := cms.Items
		sort.Slice(listOfCms, func(p, q int) bool {
			return listOfCms[p].Size() > listOfCms[q].Size()
		})

		if largestCMSize < listOfCms[0].Size() {
			largestCMSize = listOfCms[0].Size()
			largestCMName = listOfCms[0].Name
			largestCMNamespace = ns.Name
		}
		for _, item := range listOfCms {
			if item.Size() > largeSizeOfCM {
				mapOfLargestCMs[item.Name] = item.Size()
			}
		}
	}
	return largestCMSize, largestCMName, largestCMNamespace, mapOfLargestCMs, err
}

func main() {
	var kubeconfig *string
	var c conf
	configPath := c.getConf().Kubeconfigpath
	configNames, err := getKubeConfigsList(configPath)
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
		totalCRBs, err := getTotalCRBs(clientset, &ctx, &options)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("- ClusterRoleBindings Total: %d\n", totalCRBs)

		totalSecrets, err := getSecretsStats(clientset, &ctx, &options, namespaceList)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("- Secrets Total: %d\n", totalSecrets)

		roleBindingsTotal, err := getTotalRBs(clientset, &ctx, &options, namespaceList)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("- RoleBindings Total: %d\n", roleBindingsTotal)

		largestCMSize, largestCMName, largestCMNamespace, mapOfLargestCMs, err := getConfigMapStats(clientset, &ctx, &options, namespaceList)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("- Largest ConfigMap Size: %d\n", largestCMSize)
		fmt.Printf("- Largest ConfigMap Name: %s\n", largestCMName)
		fmt.Printf("- Namespace with largest ConfigMap : %s\n", largestCMNamespace)
		fmt.Println()
		fmt.Println("=== ConfigMaps Name and Size ===")
		printMap(mapOfLargestCMs)
		kubeconfig = nil
		fmt.Println()
	}
}
