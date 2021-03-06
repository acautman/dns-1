/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package options contains flags for initializing a proxy.
package options

import (
	"fmt"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	fed "k8s.io/dns/pkg/dns/federation"
)

type KubeDNSConfig struct {
	ClusterDomain      string
	KubeConfigFile     string
	KubeMasterURL      string
	InitialSyncTimeout time.Duration

	HealthzPort    int
	DNSBindAddress string
	DNSPort        int

	Federations map[string]string

	ConfigMapNs string
	ConfigMap   string

	ConfigDir    string
	ConfigPeriod time.Duration

	NameServers string
	Profiling   bool
}

func NewKubeDNSConfig() *KubeDNSConfig {
	return &KubeDNSConfig{
		ClusterDomain:      "cluster.local.",
		HealthzPort:        8081,
		DNSBindAddress:     "0.0.0.0",
		DNSPort:            53,
		InitialSyncTimeout: 60 * time.Second,

		Federations: make(map[string]string),

		ConfigMapNs: metav1.NamespaceSystem,
		ConfigMap:   "", // default to using command line flags

		ConfigPeriod: 10 * time.Second,
		ConfigDir:    "",

		NameServers: "",
	}
}

type clusterDomainVar struct {
	val *string
}

func (m clusterDomainVar) Set(v string) error {
	v = strings.TrimSuffix(v, ".")
	segments := strings.Split(v, ".")
	for _, segment := range segments {
		if errs := validation.IsDNS1123Label(segment); len(errs) > 0 {
			return fmt.Errorf("Not a valid DNS label. %v", errs)
		}
	}
	if !strings.HasSuffix(v, ".") {
		v = fmt.Sprintf("%s.", v)
	}
	*m.val = v
	return nil
}

func (m clusterDomainVar) String() string {
	return *m.val
}

func (m clusterDomainVar) Type() string {
	return "string"
}

type kubeMasterURLVar struct {
	val *string
}

func (m kubeMasterURLVar) Set(v string) error {
	parsedURL, err := url.Parse(os.ExpandEnv(v))
	if err != nil {
		return fmt.Errorf("failed to parse kube-master-url")
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" || parsedURL.Host == ":" {
		return fmt.Errorf("invalid kube-master-url specified")
	}
	*m.val = v
	return nil
}

func (m kubeMasterURLVar) String() string {
	return *m.val
}

func (m kubeMasterURLVar) Type() string {
	return "string"
}

type federationsVar struct {
	nameDomainMap map[string]string
}

func (fv federationsVar) Set(keyVal string) error {
	return fed.ParseFederationsFlag(keyVal, fv.nameDomainMap)
}

func (fv federationsVar) String() string {
	var splits []string
	for name, domain := range fv.nameDomainMap {
		splits = append(splits, fmt.Sprintf("%s=%s", name, domain))
	}
	return strings.Join(splits, ",")
}

func (fv federationsVar) Type() string {
	return "[]string"
}

func (s *KubeDNSConfig) AddFlags(fs *pflag.FlagSet) {
	fs.Var(clusterDomainVar{&s.ClusterDomain}, "domain",
		"domain under which to create names")

	fs.StringVar(&s.NameServers, "nameservers", s.NameServers,
		"List of ip:port, separated by commas of nameservers to forward queries to. "+
			"If set, overrides upstream servers taken from the nameserver option in /etc/resolv.conf. "+
			"Example: 8.8.8.8:53,8.8.4.4 (default port is 53)")

	fs.StringVar(&s.KubeConfigFile, "kubecfg-file", s.KubeConfigFile,
		"Location of kubecfg file for access to kubernetes master service;"+
			" --kube-master-url overrides the URL part of this; if this is not"+
			" provided, defaults to service account tokens")
	fs.Var(kubeMasterURLVar{&s.KubeMasterURL}, "kube-master-url",
		"URL to reach kubernetes master. Env variables in this flag will be expanded.")

	fs.IntVar(&s.HealthzPort, "healthz-port", s.HealthzPort,
		"port on which to serve a kube-dns HTTP readiness probe.")
	fs.StringVar(&s.DNSBindAddress, "dns-bind-address", s.DNSBindAddress,
		"address on which to serve DNS requests.")
	fs.IntVar(&s.DNSPort, "dns-port", s.DNSPort, "port on which to serve DNS requests.")

	fs.Var(federationsVar{s.Federations}, "federations",
		"a comma separated list of the federation names and their corresponding"+
			" domain names to which this cluster belongs. Example:"+
			" \"myfederation1=example.com,myfederation2=example2.com,myfederation3=example.com\"."+
			" It is an error to set both the federations and config-map or config-dir flags.")
	fs.MarkDeprecated("federations", "use config-dir instead. Will be removed in future version")

	fs.StringVar(&s.ConfigMapNs, "config-map-namespace", s.ConfigMapNs,
		"namespace for the config-map")
	fs.StringVar(&s.ConfigMap, "config-map", s.ConfigMap,
		"config-map name. If empty, then the config-map will not used. Cannot be "+
			"used in conjunction with federations or config-dir flag. config-map contains "+
			"dynamically adjustable configuration.")
	fs.DurationVar(&s.InitialSyncTimeout, "initial-sync-timeout", s.InitialSyncTimeout,
		"Timeout for initial resource sync.")

	fs.StringVar(&s.ConfigDir, "config-dir", s.ConfigDir,
		"directory to read config values from. Cannot be "+
			"used in conjunction with federations or config-map flag.")
	fs.DurationVar(&s.ConfigPeriod, "config-period", s.ConfigPeriod,
		"period at which to check for updates in config-dir.")
	fs.BoolVar(&s.Profiling, "profiling", s.Profiling, "specifies whether to enable profiling")
}
