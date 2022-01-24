package clusterstats

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Conf struct {
	Kubeconfigpath     string `yaml:"kubeconfigpath"`
	Configmaplargesize int    `yaml:"configmaplargesize"`
}

func (c *Conf) GetConf() *Conf {

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

func PrintMap(m map[string]int) {
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

func GetKubeConfigsList(path string) ([]string, error) {
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

func GetTotalCRBs(client *kubernetes.Clientset, ctx *context.Context, opts *v1.ListOptions) (int, error) {
	crblist, err := client.RbacV1().ClusterRoleBindings().List(*ctx, *opts)
	total := 0

	if err != nil {
		fmt.Print(err.Error())
	}
	total = len(crblist.Items)
	return total, err
}

func GetTotalRBs(client *kubernetes.Clientset, ctx *context.Context, opts *v1.ListOptions, nsList *apiv1.NamespaceList) (int, error) {
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

func GetSecretsStats(client *kubernetes.Clientset, ctx *context.Context, opts *v1.ListOptions, nsList *apiv1.NamespaceList) (int, error) {
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

func GetConfigMapStats(client *kubernetes.Clientset, ctx *context.Context, opts *v1.ListOptions, nsList *apiv1.NamespaceList) (int, string, string, map[string]int, error) {
	var c Conf
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
