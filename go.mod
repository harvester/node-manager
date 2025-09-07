module github.com/harvester/node-manager

go 1.23.6

replace (
	k8s.io/api => k8s.io/api v0.24.14
	k8s.io/apimachinery => k8s.io/apimachinery v0.24.14
	k8s.io/client-go => k8s.io/client-go v0.24.14
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.24.14
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.24.14
	k8s.io/code-generator => k8s.io/code-generator v0.24.14
	k8s.io/controller-manager => k8s.io/controller-manager v0.24.14
	k8s.io/cri-api => k8s.io/cri-api v0.24.14
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.24.14
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.24.14
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.24.14
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.24.14
	k8s.io/kubelet => k8s.io/kubelet v0.24.14
	k8s.io/kubernetes => k8s.io/kubernetes v0.24.14
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.24.14
	k8s.io/mount-utils => k8s.io/mount-utils v0.24.14
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.24.14
)

require (
	github.com/ehazlett/simplelog v0.0.0-20200226020431-d374894e92a4
	github.com/godbus/dbus/v5 v5.1.0
	github.com/harvester/go-common v0.0.0-20230718010724-11313421a8f5
	github.com/mudler/yip v1.17.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.16.0
	github.com/rancher/lasso v0.0.0-20230629200414-8a54b32e6792
	github.com/rancher/wrangler v1.1.1-0.20230818201331-3604a6be798d
	github.com/shirou/gopsutil/v3 v3.22.7
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af
	github.com/spf13/viper v1.16.0
	github.com/stretchr/testify v1.8.3
	github.com/twpayne/go-vfs v1.7.2
	github.com/urfave/cli/v2 v2.3.0
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	k8s.io/api v0.28.0
	k8s.io/apimachinery v0.28.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic v0.6.10-0.20230825194252-836f55b2639b // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/itchyny/gojq v0.12.17 // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/twpayne/go-vfs/v4 v4.3.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/oauth2 v0.13.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/term v0.29.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.30.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/code-generator v0.28.0 // indirect
	k8s.io/gengo v0.0.0-20220902162205-c0856e24416d // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9 // indirect
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
