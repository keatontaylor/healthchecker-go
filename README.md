# healthchecker-go
Example of creating custom metrics for promethus written in GoLang. This applicaiton will poll the URLs provided via command line arugments on the defined interval (see usage). After collecting the metrics it will publish them to a `/metrics` endpoint for future scraping.

# Demo
A demo of the dashboards can be found @ https://sample-grafana.invertedorigin.com. It is protected by Cloudflare Access and requires a valid @vmware.com email address. A OTP (one time password) will be sent to your @vmware.com address to reach the grafana dashbaord.

# Application Usage
```
Usage of ./healthchecker-go:
  -inverval duration
        Interval for the healthchecks (default 5s)
  -url value
        URLs to perform health checks against. Can be included multiple times for additonal URLs
```

# Prebuild Docker Image
You can run the code without compiling yourself by pulling down the latest docker image from docker hub.

```
docker pull keatontaylor/healthchecker-go:latest
```

You can then run the code via docker
```
docker run --it -p 2112:2112 keatontaylor/healthchecker-go:latest --url https://google.com --interval 60s
```

Once running you can navigate to localhost:2112/metrics to view the promethus metrics output.


The following metrics are prvided as output with descriptions and examples:
```
# HELP sample_external_url_connect_time Response time in milliseconds it took to establish the inital connection.
# TYPE sample_external_url_connect_time gauge
sample_external_url_connect_time{url="https://httpstat.us/200"} 19
sample_external_url_connect_time{url="https://httpstat.us/503"} 23
# HELP sample_external_url_dns Response time in milliseconds it took for the DNS request to take place.
# TYPE sample_external_url_dns gauge
sample_external_url_dns{url="https://httpstat.us/200"} 1
sample_external_url_dns{url="https://httpstat.us/503"} 1
# HELP sample_external_url_first_byte Response time in milliseconds it took to retrive the first byte.
# TYPE sample_external_url_first_byte gauge
sample_external_url_first_byte{url="https://httpstat.us/200"} 87
sample_external_url_first_byte{url="https://httpstat.us/503"} 97
# HELP sample_external_url_ms Response time in milliseconds it took for the URL to respond.
# TYPE sample_external_url_ms gauge
sample_external_url_ms{url="https://httpstat.us/200"} 82
sample_external_url_ms{url="https://httpstat.us/503"} 93
# HELP sample_external_url_up Status of the URL as a integer value
# TYPE sample_external_url_up gauge
sample_external_url_up{url="https://httpstat.us/200"} 1
sample_external_url_up{url="https://httpstat.us/503"} 0
```

# Building
Ensure you have a fuctional GoLang developer environmennt with version 1.17 or higher

```
git clone https://github.com/keatontaylor/healthchecker-go
go mod download
go build .
````

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

# Kube Deployments
Located in the `deployments` folder is the kube manifests for the application, they include 

# Application Overview
The applicaiton is broken into three main go routines as follows.

**main: listens for interrupt and termination signaling when running locally or within a kubernets pods and performs a graceful shutdown when applicable (interrupts)**

**httpserver: a go routine for serving the `/metrics` endpoint and watching for errors.**

**heathchecker: the major logic for performing http requests to the URls provided by your arguments (see usage). It then extracts the relevant data from the http request and sets new metrics in the prometheus client.**


