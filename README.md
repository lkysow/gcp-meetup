## Create App

```go
// cat main.go
package main

import (
	"os"
	"fmt"
	"net/http"
	"log"
)

var version = "1"
var count = make(map[string]int)

func main() {
	hostname, _ := os.Hostname()
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		greeting := r.URL.Query().Get("greeting")

		// increment the key by one
		num := count[greeting]
		count[greeting] = num + 1

		fmt.Fprintf(w, "Hello, from %s!\nI have seen that greeting %d times.\nVersion: %s\n",
			hostname,
			num,
			version,
		)
	})
	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Test it

```
go run main.go
2017/09/11 22:32:12 Starting server...
```

In a new session
```
curl localhost:8080/hello?greeting=hi
Hello, from Lukes-Macbook-Pro.local!
I have seen that greeting 0 times.
Version: 1
```

## Create Docker image

```Dockerfile
# cat Dockerfile
FROM ubuntu
ADD app .
ENTRYPOINT ["./app"]
```

Now we can build it

```
docker build .
Sending build context to Docker daemon  19.46kB
Step 1/3 : FROM scratch
 --->
Step 2/3 : ADD app .
ADD failed: stat /var/lib/docker/tmp/docker-builder297947798/app: no such file or directory
```

Ahh, of course we need to compile our app into a binary called `app`.

```
GOOS=linux go build -o app .
```

With our binary built, we can now build our Docker image

```
docker build . -t lkysow/gcp-meetup
Sending build context to Docker daemon  5.949MB
Step 1/3 : FROM ubuntu
 ---> 14f60031763d
Step 2/3 : ADD app .
 ---> 6a5687968a97
Removing intermediate container a13ad906b366
Step 3/3 : ENTRYPOINT ./app
 ---> Running in f98f82b03a69
 ---> 7a83a7a27002
Removing intermediate container f98f82b03a69
Successfully built 7a83a7a27002
Successfully tagged lkysow/gcp-meetup:latest
```

Let's run it!
```
docker run -p 8080:8080 lkysow/gcp-meetup
2017/09/12 05:45:34 Starting server...
```

And test it

```
curl localhost:8080/hello?greeting=hi
Hello, from 2f9c7131495c!
I have seen that greeting 0 times.
Version: 1
```

Now let's push up that image to the Docker registry so Kubernetes can use it

```
docker push lkysow/gcp-meetup
The push refers to a repository [docker.io/lkysow/gcp-meetup]
6be9681b14f4: Pushed
latest: digest: sha256:8e9925d8cc89a2d853d8bdfb8017eaadcb5de9cb3524484471e1c8eac4a29643 size: 528
```

## Run it in Kubernetes as a Pod

```
# cat pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: gcp-meetup
spec:
  containers:
  - image: lkysow/gcp-meetup
    name: gcp-meetup
```

Tell Kubernetes to run it

```
kubectl apply -f pod.yaml
```

Look at the logs

```
kubectl logs gcp-meetup
2017/09/12 06:07:23 Starting server...
```

Lets create a pod we can use to bounce to our Pod

```
kubectl run --image markeijsermans/debug -it debug

If you don't see a command prompt, try pressing enter.
```

Get the IP address of our pod
```
kubectl get pod -o wide
NAME                    READY     STATUS    RESTARTS   AGE       IP         NODE
debug-870039539-pqdxh   1/1       Running   0          1m        10.8.2.7   gke-cluster-1-default-pool-f2fb85a4-9n41
gcp-meetup              1/1       Running   0          3m        10.8.2.6   gke-cluster-1-default-pool-f2fb85a4-9n41
```

From the debug pod, we should be able to curl our app

```
(06:10 debug-870039539-pqdxh:/) curl 10.8.2.6:8080/hello
Hello, from gcp-meetup
I have seen that greeting 0 times.
Version: 2
```

It works!

## From Pod to Deployment
Pods aren't self-healing and we can't run multiple replicas. We need a Deployment.

```
# cat deployment.yaml
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: gcp-meetup
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: gcp-meetup
    spec:
      containers:
      - image: lkysow/gcp-meetup:v2
        name: gcp-meetup
        imagePullPolicy: Always
```

```
kubectl delete pod gcp-meetup
kubectl apply -f deployment.yaml
```

Now we have a deployment. We can delete the pod and it will come back up.

```
kubectl delete pod gcp-meetup-1664252012-kh3cb
kubectl get pod
NAME                          READY     STATUS    RESTARTS   AGE
debug-870039539-pqdxh         1/1       Running   1          28m
gcp-meetup-1664252012-kh3cb   1/1       Running   0          5s
```

We can scale the deployment
```
kubectl scale --replicas=3 deployment gcp-meetup
deployment "gcp-meetup" scaled

kubectl get pod
NAME                          READY     STATUS    RESTARTS   AGE
debug-870039539-pqdxh         1/1       Running   2          34m
gcp-meetup-1664252012-gh34p   1/1       Running   0          6s
gcp-meetup-1664252012-l07bg   1/1       Running   0          6s
gcp-meetup-1664252012-ld4zr   1/1       Running   0          5m
```

## Services
That's great but we actually want to talk to our app!

```
kubectl apply -f svc.yaml
```

Now we can curl our app with the new DNS entry and it will load balance over our Pods.

```
curl gcp-meetup/hello?greeting=bye
Hello, from gcp-meetup-1883113546-qc4gz!
I have seen that greeting 1805 times.
Version: 2
```

## Redis
You might notice that each pod is keeping track of a different count. Let's push that
state out to Redis instead.

We need to create a new Redis `Service` and `Deployment`.

```
# cat redis.yaml
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: redis
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - image: redis
        name: redis
---
kind: Service
apiVersion: v1
metadata:
  name: redis
spec:
  selector:
    app: redis
  ports:
  - protocol: TCP
    port: 6379
    targetPort: 6379
```

Test that redis is running from our debug pod

```
nc redis 6379
PING
+PONG
```

Now we can call redis from our app

```go
import (
    "os"
    "fmt"
    "net/http"
    "log"
    "github.com/go-redis/redis"
)

var version = "2"
func main() {
	hostname, _ := os.Hostname()
	client := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
	})

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		greeting := r.URL.Query().Get("greeting")

		// increment the key by one
		num, err := client.Incr(greeting).Result()
		if err != nil {
			w.WriteHeader(503)
			fmt.Fprintf(w, err.Error())
		}

		fmt.Fprintf(w, "Hello, from %s!\nI have seen that greeting %d times.\nVersion: %s\n",
			hostname,
			num,
			version,
		)
	})
	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Compile the golang app, build a new docker image, tag it, and push it

```
GOOS=linux go build -o app .
docker build -t lkysow/gcp-meetup:v2 .
docker push lkysow/gcp-meetup:v2
```

Edit our `deployment.yaml` to use the `v2` image and then run

```
kubectl apply -f deployment.yaml
```

We should see a rolling deploy succeed and now the count should be the same across pods.
