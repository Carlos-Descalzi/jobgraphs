TAG?=latest
IMAGE_NAME=carlosdescalzi/jobgraphs:$(TAG)

run: 
	go run cmd/* -kubeconfig=$(KUBECONFIG) 

build: 
	docker build -t $(IMAGE_NAME) $(PWD)