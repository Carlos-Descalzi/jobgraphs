# jobgraphs
Kubernetes controller for running jobs specified in a directed acyclic graph definition.
Jobs represent nodes in the graph, when the resource is created, the controller will start running jobs according to the graph definition.
The field 'jobFailureCondition' in job graph has two possible values, fail or continue, this tells the controller what to do once a job/node fails. 
The field 'graph' declares the graph definition as an array of edges, defined by source and target job names.

Example jobgraph:
```yaml
apiVersion: ced.io/v1alpha1
kind: JobGraph
metadata:
  name: test-graph
spec:
  jobFailureCondition: fail
  graph:
    edges:
      - source: job1
        target: job2
  jobTemplates:
    - metadata:
        name: job1
      spec:
        template:
          spec:
            containers:
              - image: ubuntu:20.04
                command: ["echo","hello world"]
            restartPolicy: Never
    - metadata:
        name: job2
      spec:
        template:
          spec:
            containers:
              - image: ubuntu:20.04
                command: ["echo","hello world"]
            restartPolicy: Never
```

## Install 
```bash
kubectl apply -f https://github.com/Carlos-Descalzi/jobgraphs/blob/main/deploy/deploy.yaml
```
