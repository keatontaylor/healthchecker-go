# healthchecker-go
Example of creating custom metrics for promethus written in GoLang. This applicaiton will poll the URLs provided via command line arugments on a defined interval (see [usage](#application-usage)). 

Once metrics on the URL request have been collected, it will set the metrics via the prometheus GoLang Library. At this point they will be available for scapring at `/metrics` on port `2112`

# Demo
A demo hosted on a Linode k8s cluster can be found @ https://sample-grafana.invertedorigin.com. 

To reach the dashboard please do the following.
1. Navigate to the above URL and enter your @vmware.com email address.
2. A OTP (one time password) will be sent to your email, retrieve it and type it into the box provided.
3. Once at the grafana homepage naviagate to Dashboards -> Browse. Then select the dashboard titled "HeathChecker".

### Demo Architecture
The demo is hosted on LKE (Linode Kubernetes Environment). This environment has loki-stack (Loki, Promtail, Grafana, Prometheus) installed and also includes a cloudfalred client.

Requsets to the above URL are proxied by Cloudflare, where they are subject to Cloudflare Zero Trust rules. Once a policy has been met (providing the OTP from your email) requests are then forwarded to an highly available cloudflared deployment within the LKE cluster. 

Cloudflared is configured with a named tunnel between Cloudflare's Edge Network and the LKE cluster. As a result it is effectively operating as a HA reverse proxy negating the need for an ingress controller or load balancer. All public requests are load balanced (round-robin) by the cloudflared deployment.

# Running Locally
### Via the Prebuild Docker Image
```
docker pull keatontaylor/healthchecker-go:latest

docker run -it --rm -p 2112:2112 keatontaylor/healthchecker-go:latest --url https://httpstat.us/503 --url https://httpstat.us/200 --interval 5s
```

### Via Local Building
Ensure you have a fuctional GoLang developer environmennt with version 1.17 or higher

```
git clone https://github.com/keatontaylor/healthchecker-go
go mod download
go build .
```
#### Application Usage
```
Usage of ./healthchecker-go:
  -inverval duration
        Interval for the healthchecks (default 5s)
  -url value
        URLs to perform health checks against. Can be included multiple times for additonal URLs
```

# Testing
Test cases have been written to validate the request and output of the functions. See the following 
```
go test ./... -v 
```
Example output:
```
=== RUN   TestFetchStats200
2022/03/16 21:19:02 Updating custom metrics: url: http://127.0.0.1:56069/200, connectMS: 0, dnsMS: 0, firstbyteMS: 1, totalMS: 0, status: 1
--- PASS: TestFetchStats200 (0.00s)
=== RUN   TestFetchStats503
2022/03/16 21:19:02 Updating custom metrics: url: http://127.0.0.1:56071/503, connectMS: 0, dnsMS: 0, firstbyteMS: 0, totalMS: 0, status: 0
--- PASS: TestFetchStats503 (0.00s)
=== RUN   TestUpdateCustomMetrics
2022/03/16 21:19:02 Updating custom metrics: url: testurl.com, connectMS: 4, dnsMS: 2, firstbyteMS: 3, totalMS: 9, status: 1
--- PASS: TestUpdateCustomMetrics (0.00s)
PASS
ok      github.com/keatontaylor/healthchecker-go/pkg/healthchecker      (cached)
```

# Building and Pushing to Docker
The included Dockerfile can be used to build the applciation for multiple platforms using the buildx process. 

```
docker buildx create
docker buildx build --platform linux/amd64,linux/arm64 -t <dockerhub username>/healthchecker-go:latest .
```

# Kube Deployments
Located in the `deployments` folder is the kube manifest for the application, the `deployment.yaml` file contains the namespace, deployment and service definitions along with all necessary annotations for the promethus service scrape config.

# Deploying the entire stack locally
Requries the installation of either minikube or docker desktop kubeernetes.

Other requirements include:
1. Helm installed
2. Helm repo for grafana stack `helm repo add grafana https://grafana.github.io/helm-charts`
3. kube context set to the minikube or docker desktop installation.


## Deploy applicaiton
```
kubectl apply -f deployments/deployment.yaml
```

## Deploy Loki Stack 
The command below will deploy loki, grafana, promethus and promtail. By default it will have no peristence as it is not reuqired in a local testing environment.
```
helm upgrade --install loki grafana/loki-stack  --set grafana.enabled=true,prometheus.enabled=true,prometheus.alertmanager.persistentVolume.enabled=false,prometheus.server.persistentVolume.enabled=false,loki.persistence.enabled=false
```

## Fetching the grafana password
Run the command to get the base64 decoded admin password and username for the local grafana deployment.
```
kubectl get secret loki-grafana -o json | jq '.data | map_values(@base64d)'
```

## Forwarding Grafana Port
Using typical port forwarding commands. The full pod name can be found via `kubectl get pods`
```
kubectl port-forward loki-grafana-<replace with pod id> 3000:3000
```

## Accessing the grafana UI and uploading the dashboard
1. Naviate to localhost:3000 in your browser and use the username and password retrieved from the prior steps.
2. Navigate to Dashboards -> Browse
3. Click `Import`
4. Copy and paste the `dashboard.json` file located in the `deployments` directory.
5. Hitting `Import` will take you to the dashboard.
