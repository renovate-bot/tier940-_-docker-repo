package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type HostInfo struct {
	Host string `json:"host"`
	Path string `json:"path"`
}

func getIngressHosts(clientset *kubernetes.Clientset, excludeSelf bool) ([]HostInfo, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var hosts []HostInfo
	selfHostname := os.Getenv("SELF_HOSTNAME") // 自身のホスト名

	for _, ingress := range ingresses.Items {
		for _, rule := range ingress.Spec.Rules {
			// 自身を除外する処理
			if excludeSelf && strings.EqualFold(rule.Host, selfHostname) {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				hosts = append(hosts, HostInfo{
					Host: rule.Host,
					Path: path.Path,
				})
			}
		}
	}
	return hosts, nil
}

func main() {
	// Kubernetes APIクライアントのセットアップ
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error building in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	// HTTP サーバーのセットアップ
	http.HandleFunc("/hosts", func(w http.ResponseWriter, r *http.Request) {
		excludeSelf := r.URL.Query().Get("excludeSelf") == "true"
		hosts, err := getIngressHosts(clientset, excludeSelf)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching Ingress hosts: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hosts)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}