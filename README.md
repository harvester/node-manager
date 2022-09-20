Harvester Node Manager
========

A node manager helps to manage the host kernel configuration of the [Harvester](https://github.com/harvester/harvester) cluster, eg: KSM.

## Manifests and Deploying
The `./manifests` folder contains useful YAML manifests to use for deploying and developing the Harvester node manager. 
This simply YAML deployment creates a Daemonset using the `rancher/harvester-node-manager` container.

## License
Copyright (c) 2022 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.