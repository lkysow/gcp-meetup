## Create App

```go
// main.go
package main

import (
	"os"
	"fmt"
	"net/http"
	"log"
)

var version = "1"

func main() {
	hostname, _ := os.Hostname()
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, from %s! My version is %s, our config is %q and our secret is %q",
			hostname,
			version,
			os.Getenv("CONFIG"),
			os.Getenv("SECRET"),
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
curl localhost:8080/hello
Hello, from Lukes-Macbook-Pro.local! Our config is "" and our secret is ""
```

## Create Docker image

```Dockerfile
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
curl localhost:8080/hello
Hello, from 0bceb5c0aa5f! Our config is "" and our secret is ""%
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
# pod.yaml
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
Hello, from gcp-meetup! Our config is "" and our secret is ""
```

It works!

## ConfigMaps

We want to use the same container image for every environment, but often configuration
changes per environment. That's where `ConfigMap`'s come in.

```
# configmap.yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: gcp-meetup
data:
  date: sep14
```

Create it

```
kubectl apply -f configmap.yaml
configmap "gcp-meetup" created
```

Now lets get our pod to use the `ConfigMap` via an environment variable.
Delete the running pod and create a new one

```
apiVersion: v1
kind: Pod
metadata:
  name: gcp-meetup
spec:
  containers:
  - image: lkysow/gcp-meetup
    name: gcp-meetup
    env:
    - name: CONFIG
      valueFrom:
        configMapKeyRef:
          # The ConfigMap containing the value you want to assign to SPECIAL_LEVEL_KEY
          name: gcp-meetup
          # Specify the key associated with the value
          key: date
```

Now when we curl the Pod we should see the date in the response

```
kubectl attach -it debug-870039539-pqdxh
If you don't see a command prompt, try pressing enter.

(06:18 debug-870039539-pqdxh:/) curl 10.8.2.8:8080/hello
Hello, from gcp-meetup! Our config is "sep14" and our secret is ""
```

## Secrets

Create a secret file on disk

```
echo hunter2 > secret.txt
```

Use `kubectl` to create the secret
```
kubectl create secret generic gcp-secret --from-file=password=./secret.txt
secret "gcp-secret" created
```

Check that it's there

```
kubectl describe secret gcp-secret
Name:		gcp-secret
Namespace:	default
Labels:		<none>
Annotations:	<none>

Type:	Opaque

Data
====
password:	7 bytes
```

Note that you can't see the secret unless you use `get -o yaml`

Now let's get the Pod to use the secret as an environment variable

```
apiVersion: v1
kind: Pod
metadata:
  name: gcp-meetup
spec:
  containers:
  - image: lkysow/gcp-meetup
    name: gcp-meetup
    env:
    - name: CONFIG
      valueFrom:
        configMapKeyRef:
          name: gcp-meetup
          key: date
    # Secret below
    - name: SECRET
      valueFrom:
        secretKeyRef:
          name: gcp-secret
          key: password
```

Delete the old Pod, create a new one and see if it worked.
```
curl 10.8.2.10:8080/hello
Hello, from gcp-meetup! Our config is "sep13" and our secret is "hunter2"
```

## From Pod to Deployment

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
